+++
title = "Writing a Reflective PE Loader in 2024 for the Xbox"
description = "Adventures in reinventing the wheel"
summary = "Adventures in reinventing the wheel"
template = "toc_page.html"
toc = true
date = "2024-07-28"

[extra]
#image = "/img/wows-obfuscation/header.png"
#image_width =  250
#image_height = 250
pretext = """
Adventures in reinventing the wheel
"""
+++

Recently a good friend of mine, Emma ( [@carrot_c4k3](https://twitter.com/carrot_c4k3)), participated in pwn2own in the Windows LPE category and ended up using a great bug for LPE. The bug far exceeded the basic category though: this vulnerability was also a _sandbox escape_, i.e. it's in an NT syscall which is reachable from the UWP sandbox. Her and I met about 17 years ago in the Xbox 360 hacking scene and she got a wild idea: why not try to port the exploit over to the Xbox One?

## Brief Primer of Xbox One's Security

Since I'll be talking about this in the context of the Xbox, it's worthwhile to spend a moment discussing the Xbox One's security model. There's [a very great and in-depth overview of the Xbox One's security model on YouTube](https://www.youtube.com/watch?v=U7VwtOrwceo) presented by Tony Chen who is one of the folks who designed it. I highly recommend watching it if you're interested, but I'll do my best at giving a crash course:

```
┌────────────────────────────┐     ┌────────────────────────────┐
│                            │     │                            │
│                            │     │                            │
│                            │     │                            │
│                            │     │                            │
│        ERA (GameOS)        │     │         SystemOS           │
│                            │     │                            │
│                            │     │                            │
│                            │     │     │             │        │
│                            │     │     │             │        │
└────────────────────────────┘     └─────┼─────────────┼┬───────┤
               │                         │             ││ VMBus │
               │                         │             │└───────┘
┌──────────────┼─────────────────────────┼─────────────┼────────┐
│              │                         ▼             ▼        │
│              │           HostOS  ┌──────────┐ ┌───────────────┤
│              │                   │ Synthetic│ │ VSPs/Normal   │
│              └──────────────────▶│ Devices  │ │ Hyper-V Stuff │
└──────────────────────────────────┴──────────┴─┴───────────────┘
┌───────────────────────────────────────────────────────────────┐
│                                                               │
│                          Hypervisor                           │
│                                                               │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

This is a very, very simplified drawing of what you'd find on Microsoft's [Hyper-V Architecture page](https://learn.microsoft.com/en-us/virtualization/hyper-v-on-windows/reference/hyper-v-architecture). The main thing I'm trying to highlight here is that there are 3 VMs with 3 different purposes:

1. HostOS, which acts very similar to your standard Hyper-V host
2. ERA OS (aka GameOS) which is where games run
3. SystemOS which is where applications run.

All of the operating systems for each VM are a very slimmed down version of Windows based on Windows Core OS (WCOS) and the Hyper-V architecture is mostly what you'd encounter on a normal PC.

There's something missing here though, which is the _security processor_ (SP). The Xbox One's security processor should be the only thing on the Xbox which can reveal a title's plaintext on Xbox One. Microsoft's [Pluton Processor](https://learn.microsoft.com/en-us/windows/security/hardware-security/pluton/microsoft-pluton-security-processor) is based on learnings from the Xbox One's security processor.

The core idea behind all of this is to **make piracy extremely difficult** (if not impossible without breaking the SP) and to make it so that if you _do_ hack the Xbox One, you can't do it online trivially because the SP will attest that the console's state is something unexpected.

## OK, How Does This Relate to the PE loader?

Emma found a vulnerability/feature in an application on the Xbox One marketplace called _GameScript_, which is an ImGui UI for messing with the [Ape programming language](https://github.com/kgabis/ape). Through this vulnerability Emma was able to read/write arbitrary memory and run shellcode. But now the problem: how can we run arbtirary executables easily?

Since we have the ability to read/write arbitrary memory and change page permissions, Emma asked if I would write a PE loader to simplify the development pipeline while she worked on porting her exploit over. Easy enough right?

**Wrong.**

## Reinventing the Wheel

The specific technique of user-mode portable executable (PE/.exe file) loading is referred as "Reflective PE Loading" which to me is some #redteam term I'd never heard before embarking on this project. This name is meaningless in my opinion, but a "reflective PE loader" is simply some user-mode code that can load a PE without going through the normal `LoadLibrary()` / `CreateProcess()` routines and executes the PE's entrypoint. I took a look at the work involved and decided I was better off writing my own loader for a few reasons:

1. I despise dealing with C/C++ build systems
2. Since I'm targeting _Xbox_ and not _Windows_, I might encounter some problems and I know how to debug my own code better than someone else's
3. On the Xbox we're going to _have to_ use a PE loader for running any executable until we eventually break code integrity. So we better know how it works and be able to load complex applications.
4. I don't give a shit about EDR evasion or anything like that.
5. It seemed simple enough at the time to just rewrite it in Rust, so I did.

For my project's base I combined two open-source Rust projects:

- [b1tg/rust-windows-shellcode](https://github.com/b1tg/rust-windows-shellcode) which provided a great template for writing and building Windows shellcode in Rust.
- [Thoxy67/rspe](https://github.com/Thoxy67/rspe) which provides a basic reflective loader.

rspe already got me most of the way there, but with a few caveats:

- It did not support loading imports by ordinal
- Thoxy67 got great work done, but it seemed like it was written by someone newer to Rust and needed some cleanup (e.g. unnecessary copies)
- It did not support thread-local storage at all
- It did not support command line arguments
- It did not support environments with W^X mitigations
- It did not work with _shellcode-based programming_ in mind.

And on that last point, you might be wondering, "What is shellcode-based programming?" Well why don't I just give an example. Here's how a `VirtualAlloc()` call in `rspe` worked before:

```rust
#[link(name = "kernel32")]
extern "system" {
    pub fn VirtualAlloc(
        lpaddress: *const c_void,
        dwsize: usize,
        flallocationtype: VIRTUAL_ALLOCATION_TYPE,
        flprotect: PAGE_PROTECTION_FLAGS,
    ) -> *mut c_void;
}

// Allocate memory for the image
let baseptr = VirtualAlloc(
    core::ptr::null_mut(), // lpAddress: A pointer to the starting address of the region to allocate.
    imagesize,             // dwSize: The size of the region, in bytes.
    MEM_COMMIT,            // flAllocationType: The type of memory allocation.
    PAGE_EXECUTE_READWRITE, // flProtect: The memory protection for the region of pages to be allocated.
);
```

And here's how this would look with shellcode-based programming:

```rust
pub type VirtualAllocFn = unsafe extern "system" fn(
    lpAddress: *const c_void,
    dwSize: usize,
    flAllocationType: u32,
    flProtect: u32,
) -> PVOID;

pub fn fetch_virtual_alloc(kernelbase_ptr: PVOID) -> VirtualAllocFn {
    // this is some macro that, using `kernelbase_ptr`, parses kernelbase's export table to find `VirtualAlloc`
    // and return its address. i.e. kind of a self-made version of `GetProcAddress`
    resolve_func!(kernelbase_ptr, "VirtualAlloc")
}

let VirtualAlloc = fetch_virtual_alloc(kernelbase_ptr);
let baseptr = (VirtualAlloc)(
    preferred_load_addr, // lpAddress: A pointer to the starting address of the region to allocate.
    imagesize,           // dwSize: The size of the region, in bytes.
    MEM_COMMIT,          // flAllocationType: The type of memory allocation.
    PAGE_READWRITE,     // flProtect: The memory protection for the region of pages to be allocated.
);
```

As you might have noticed, we're not linking against any libraries and calling imported functions directly. Instead we're using indirect calls based off of resolving function addresses at runtime.

## The Easy Parts

Although it's been talked about before, I'll give a brief overview of how a basic loader works:

1. Parse the PE headers and `VirtualAlloc()` some memory for the "cloned" PE with all the fixups applied. You'll try to `VirtualAlloc()` at the PE's preferred load address, but if you don't get it fall back to a random address -- this is your _load address_. From here you calculate the delta between the preferred and actual load address and this will be used for fixing relocations.
2. Iterate each PE section and copy it over to the newly `VirtualAlloc`'d region. The virtual addresses here are _relative_ virtual addresses, so you just take each section's VirtualAddress, add it to the load address. Copy the section from its old location to the new address.
3. For each section, look at its `Characteristics` field and determine the correct permissions. `VirtualProtect()` the section according to the permissions.
4. For each import in the import table (`IMAGE_DIRECTORY_ENTRY_IMPORT`), ensure the imported DLL is loaded, then use the loaded DLL's handle with `GetProcAddress()` to get the address of the function being loaded. You could also parse the module's exports and match things up, but this is the lazy way. For each import, overwrite the real address in the import's thunk.
5. Fix relocations. This basically involves walking the `IMAGE_DIRECTORY_ENTRY_BASERELOC` directory and fixing each `IMAGE_BASE_RELOCATION` such that you add the delta calculated in step 1 to the relocation's `VirtualAddress` field. There's some nuance here where you need to only modify certain bits, etc. etc. but this is the basic idea.
6. Call the module's entrypoints.

Step 6 is actually a bit of a heavy bullet point. I learned through this experience that PEs can actually have _multiple_ thread-local storage callbacks called before the actual module entrypoint. Calling these is fairly straightforward:

```rust
let tls_directory =
    &ntheader_ref.OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_TLS as usize];

// Grab the TLS data from the PE we're loading
let tls_data_addr =
    baseptr.offset(tls_directory.VirtualAddress as isize) as *mut IMAGE_TLS_DIRECTORY64;

let tls_data: &mut IMAGE_TLS_DIRECTORY64 = unsafe { core::mem::transmute(tls_data_addr) };

let mut callbacks_addr = tls_data.AddressOfCallBacks as *const *const c_void;
if !callbacks_addr.is_null() {
    let mut callback = unsafe { *callbacks_addr };

    while !callback.is_null() {
        execute_tls_callback(baseptr, callback);
        callbacks_addr = callbacks_addr.add(1);
        callback = unsafe { *callbacks_addr };
    }
}
```

Then you call the actual module's entrypoint:

```rust
unsafe fn execute_tls_callback(baseptr: *const c_void, entrypoint: *const c_void) {
    let func: ImageTlsCallbackFn = core::mem::transmute(entrypoint);
    func(baseptr, DLL_THREAD_ATTACH, ptr::null_mut());
}
```

Executing the image entrypoint is pretty similar:

```rust
#[cfg(target_arch = "x86_64")]
let entrypoint = (baseptr as usize
    + (*(ntheader as *const IMAGE_NT_HEADERS64))
        .OptionalHeader
        .AddressOfEntryPoint as usize) as *const c_void;
#[cfg(target_arch = "x86")]
let entrypoint = (baseptr as usize
    + (*(ntheader as *const IMAGE_NT_HEADERS32))
        .OptionalHeader
        .AddressOfEntryPoint as usize) as *const c_void;

// Create a new thread to execute the image
execute_image(baseptr, entrypoint, context.fns.create_thread_fn);

unsafe fn execute_image(
    dll_base: *const c_void,
    entrypoint: *const c_void,
    create_thread_fn: CreateThreadFn,
) {
    let func: extern "system" fn(*const c_void, u32, *const c_void) -> u32 =
        core::mem::transmute(entrypoint);
    func(dll_base, DLL_PROCESS_ATTACH, ptr::null());
}
```

## The Hard Parts

There were some parts that really kicked my ass in figuring out, but in my opinion were very important for what I wanted in the PE loader. The big two to me were:

1. The exploit / PE loader must not cause the host application to become unreliable. I don't want to be debugging crashes in some of the "host application" threads that broke simply because we're hijacking its address space.
2. We must be able to run complex applications. Since we're using this technique to bypass code integrity, this will be our main method of running arbitrary applications.
3. The application shouldn't _know_ it's been reflectively loaded, or care.

### Thread-Local Storage

Related to #2, the absolute biggest challenge I faced was with applications that use thread-local storage. Having done all of my development in Rust, my test program that I was loading was also obviously in Rust. I kept encountering `int 29` (`RtlFailFast(code)`) instructions that crashed the application though when I'd execute its entrypoint. This was **extremely** painful to debug, but eventually I figured out that I was failing after fetching data from thread-local storage:

[![Screenshot of assembly instructions from a Rust "hello world" application loading data from thread-local storage in IDA pro](/img/pe-loader/tls-thread-set-current.png)](/img/pe-loader/tls-thread-set-current.png)

[![Screenshot of assembly instructions from a Rust "hello world" application executing an `int 29` instruction in IDA pro](/img/pe-loader/int-29.png)](/img/pe-loader/int-29.png)

I was kind of confused because I didn't expect my application to use thread-local storage, but even a basic "hello world" Rust program did:

[![Screenshot of a Rust "hello world" application loaded into the "PE Bear" program, showing its TLS directory](/img/pe-loader/pe-bear-tls.png)](/img/pe-loader/pe-bear-tls.png)

It turns out that this is related to Rust's thread initialization code that sets some thread-locals for the current thread and thread ID: https://github.com/rust-lang/rust/blob/2e630267b2bce50af3258ce4817e377fa09c145b/library/std/src/thread/mod.rs#L694

So I come to realize that my original idea for how I was handling thread-local storage was completely flawed. I was _allocating_ new memory for my module's TLS, but didn't even realize it had some default state associated with it that I had to copy over. Simple fix right?

```patch
diff --git a/crates/loader/src/lib.rs b/crates/loader/src/lib.rs
index 97311d0..d66773d 100755
--- a/crates/loader/src/lib.rs
+++ b/crates/loader/src/lib.rs
@@ -180,34 +185,53 @@ unsafe fn reflective_loader_impl(context: LoaderContext) {
             .OptionalHeader
             .AddressOfEntryPoint as usize) as *const c_void;

-    let tls_directory = &ntheader_ref.OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_TLS];
+    let tls_directory =
+        &ntheader_ref.OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_TLS as usize];
+
+    // Grab the TLS data from the PE we're loading
+    let tls_data_addr =
+        baseptr.offset(tls_directory.VirtualAddress as isize) as *mut IMAGE_TLS_DIRECTORY64;
+
+    // TODO: Patch the module list
+    let tls_index = patch_module_list(
+        context.image_name,
+        baseptr,
+        imagesize,
+        context.fns.get_module_handle_fn,
+        tls_data_addr,
+        context.fns.virtual_protect,
+        entrypoint,
+    );
+
     if tls_directory.Size > 0 {
         // Grab the TLS data from the PE we're loading
         let tls_data_addr =
             baseptr.offset(tls_directory.VirtualAddress as isize) as *mut IMAGE_TLS_DIRECTORY64;

-        let tls_data: &IMAGE_TLS_DIRECTORY64 = unsafe { core::mem::transmute(tls_data_addr) };
+        let tls_data: &mut IMAGE_TLS_DIRECTORY64 = unsafe { core::mem::transmute(tls_data_addr) };

         // Grab the TLS start from the TEB
         let tls_start: *mut *mut c_void;
         unsafe { core::arch::asm!("mov {}, gs:[0x58]", out(reg) tls_start) }

-        let tls_index = unsafe { *(tls_data.AddressOfIndex as *const u32) };
-
         let tls_slot = tls_start.offset(tls_index as isize);
         let raw_data_size = tls_data.EndAddressOfRawData - tls_data.StartAddressOfRawData;
-        *tls_slot = (context.fns.virtual_alloc)(
+        let tls_data_addr = (context.fns.virtual_alloc)(
             ptr::null(),
-            raw_data_size as usize,
+            raw_data_size as usize, // + tls_data.SizeOfZeroFill as usize,
             MEM_COMMIT,
             PAGE_READWRITE,
         );

-        // if !tls_start.is_null() {
-        //     // Zero out this memory
-        //     let tls_slots: &mut [u64] = unsafe { core::slice::from_raw_parts_mut(tls_start, 64) };
-        //     tls_slots.iter_mut().for_each(|slot| *slot = 0);
-        // }
+        core::ptr::copy_nonoverlapping(
+            tls_data.StartAddressOfRawData as *const _,
+            tls_data_addr,
+            raw_data_size as usize,
+        );
+
+        // Update the TLS index
+        core::ptr::write(tls_data.AddressOfIndex as *mut u32, tls_index);
+        *tls_slot = tls_data_addr;

         let mut callbacks_addr = tls_data.AddressOfCallBacks as *const *const c_void;
         if !callbacks_addr.is_null() {
```

This code *worked*, but it didn't work for long. I obviously had no idea how thread-local storage worked, and soon discovered that in my multi-threaded application I was _again_ getting similar crashes because the TLS data was bad. Through much pain and debugging I ended up learning:

1. Changing the thread-local storage for your current thread is obviously not enough. New threads that spawn won't have the modifications I did above, so they'll have "default" TLS without my module included since the changes I did above are only reflected for the current thread. Duh.
2. TLS is allocated in slots for the current thread and each slot is a pointer to the TLS data.
3. Windows keeps a cache of TLS directories for each loaded module, which makes solving the above for new threads pretty challenging.

#### Patching ntdll for One Stupid List

In the above section I mentioned that Windows keeps a cache of TLS directories for each loaded module, and I think this is a critical reason why the reflective PE loaders  I sampled didn't bother with TLS data. I really only discovered this by painful debugging and figuring out the application only crashed when spawning new threads, that the crashes were relating to TLS, and figuring that something must be wrong with the TLS data. It finally clicked when I noticed that the `ThreadLocalStoragePointer` for the crashing thread's TEB didn't match the spawning thread's...

[![!teb command in WinDbg](/img/pe-loader/teb-command.png)](/img/pe-loader/teb-command.png)

[![Clicking the TEB pointer in WinDbg's !teb output](/img/pe-loader/thread-local-storage.png)](/img/pe-loader/thread-local-storage.png)

This is super obvious in hindsight! The TLS has to be unique, but I don't know... I thought the `ThreadLocalStoragePointer` was a pointer to the _default state_ TLS and the per-thread slots were in the TEB's `TlsSlots` field?

Anyways, I set a breakpoint at the thread initialization routine, `LdrpInitializeThread`, and debugged it to see if there was anything that stood out for TLS initialization. Like magic, I eventually stepped into `LdrpAllocateTls`:

[![WinDbg stack for a new user thread showing the call into LdrpAllocateTls](/img/pe-loader/LdrpAllocateTls.png)](/img/pe-loader/LdrpAllocateTls.png)

The [ReactOS source code](https://github.com/mirror/reactos/blob/c6d2b35ffc91e09f50dfb214ea58237509329d6b/reactos/dll/ntdll/ldr/ldrinit.c#L1215-L1273) was of huge help here in figuring out what was going on, but essentially what happens when spawning a new thread is:

1. If any of the currently loaded modules has TLS, allocate a `ThreadLocalStoragePointer`.
2. The size of this memory block is `sizeof(void*) * NUM_MODULES_WITH_TLS_DATA`
3. Iterate some `TlsLinks` list. This is a list of `LDRP_TLS_DATA`:

```c
typedef struct _LDRP_TLS_DATA
{
    LIST_ENTRY TlsLinks;
    IMAGE_TLS_DIRECTORY TlsDirectory;
} LDRP_TLS_DATA, *PLDRP_TLS_DATA;
```

4. Calculate the size of the TLS data based on the `TlsDirectory`, and copy its contents
5. Put the pointer to the memory allocated in step 4 in the appropriate slot, recorded as `TlsData->TlsDirectory.Characteristics`.

Now that I know the TLS data is cached, can't I just overwrite the `TlsDirectory` data in this list from the host module with the data from the new module? Well yes... and no. The `LDRP_TLS_DATA` is heap-allocated, so I'd have to scan the heap which would be pretty bug-prone.

Instead I popped `ntdll.dll` into IDA to see what functions were using this `LdrpTlsList` to see if maybe there was some other way I could grab the list's address.

[![IDA Pro window showing functions using LdrpTlsList](/img/pe-loader/LdrpFindTlsEntry.png)](/img/pe-loader/LdrpFindTlsEntry.png)

I conveniently found that in Windows (but not ReactOS) is a function, "LdrpFindTlsList", which will return a `PTLS_ENTRY` (the actual Windows data structure for `LDRP_TLS_DATA`) given a `PLDR_DATA_TABLE_ENTRY`. Ken Johnson even conviently provided the source code on his blog: http://www.nynaeve.net/Code/VistaImplicitTls.cpp

So now the only missing link: finding the `PLDR_DATA_TABLE_ENTRY`. This actually wasn't so bad and I needed to find this anyways in order to patch command line arguments and image name:

```rust
pub unsafe fn patch_module_list(
    image_name: Option<&[u16]>,
    new_base_address: *mut c_void,
    module_size: usize,
    get_module_handle_fn: GetModuleHandleAFn,
    this_tls_data: *const IMAGE_TLS_DIRECTORY64,
    virtual_protect: VirtualProtectFn,
    entrypoint: *const c_void,
) -> u32 {
    let current_module = get_module_handle_fn(core::ptr::null());

    let teb = teb();
    let peb = (*teb).ProcessEnvironmentBlock;
    let ldr_data = (*peb).Ldr;
    let module_list_head = &mut (*ldr_data).InMemoryOrderModuleList as *mut LIST_ENTRY;
    let mut next = (*module_list_head).Flink;
    while next != module_list_head {
        // -1 because this is the second field in the LDR_DATA_TABLE_ENTRY struct.
        // the first one is also a LIST_ENTRY
        let module_info = (next.offset(-1)) as *mut LDR_DATA_TABLE_ENTRY;
        if (*module_info).DllBase == current_module {
            (*module_info).DllBase = new_base_address;
            // EntryPoint
            (*module_info).Reserved3[0] = entrypoint as *mut c_void;
            // SizeOfImage
            (*module_info).Reserved3[1] = module_size as *mut c_void;

            if !this_tls_data.is_null() {
                let ntdll_addr = get_module_handle_fn("ntdll.dll\0".as_ptr() as *const _);
                if let Some(ntdll_text) = get_module_section(ntdll_addr as *mut _, b".text") {
                    for window in ntdll_text.windows(LDRP_FIND_TLS_ENTRY_SIGNATURE_BYTES.len()) {
                        if window == LDRP_FIND_TLS_ENTRY_SIGNATURE_BYTES {
                            // Get this window's pointer and move backwards to find the start of the fn
                            let mut ptr = window.as_ptr();
                            loop {
                                let behind = ptr.offset(-1);
                                if *behind == 0xcc {
                                    break;
                                }
                                ptr = ptr.offset(-1);
                            }

                            let LdrpFindTlsEntry: LdrpFindTlSEntryFn = core::mem::transmute(ptr);

                            let list_entry = LdrpFindTlsEntry(module_info);

                            (*list_entry).TlsDirectory = *this_tls_data;
                        }
                    }
                }
            }
            break;
        }
        next = (*next).Flink;
    }

    if !this_tls_data.is_null() {
        let dosheader = get_dos_header(current_module);
        let ntheader = get_nt_header(current_module, dosheader);

        #[cfg(target_arch = "x86_64")]
        let ntheader_ref: &mut IMAGE_NT_HEADERS64 = unsafe { core::mem::transmute(ntheader) };
        #[cfg(target_arch = "x86")]
        let ntheader_ref: &mut IMAGE_NT_HEADERS32 = unsafe { core::mem::transmute(ntheader) };

        let real_module_tls_entry =
            &mut ntheader_ref.OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_TLS as usize];

        let real_module_tls_dir = current_module
            .offset(real_module_tls_entry.VirtualAddress as isize)
            as *mut IMAGE_TLS_DIRECTORY64;

        let mut old_perms = 0;
        virtual_protect(
            real_module_tls_dir as *mut _ as *const _,
            core::mem::size_of::<IMAGE_TLS_DIRECTORY64>(),
            PAGE_READWRITE,
            &mut old_perms,
        );

        let idx = *((*real_module_tls_dir).AddressOfIndex as *const u32);
        *real_module_tls_dir = *this_tls_data;

        idx
    } else {
        0
    }
}
```

### Patching Command-Line Args and Image Name

This has been done by other PE loaders, but I wanted to call this out as well: while the PEB contains the image name and process arugments, so does `kernelbase.dll`! This one wasn't _too_ bad to patch so long as you want to rely on the fact that the `UNICODE_STRING` structure for the PEB and in `kernelbase.dll` share the same backing buffer (i.e. the latter is a shallow copy of the former). That also doesn't account for the `ANSI_STRING` variant... but 🤷‍♂️

tl;dr of this code: we scan the global memory of `kernelbase.dll` looking for the previously mentioned pointer, and when we find it we just update its pointer and length to match our new pointer and length.

```rust
pub unsafe fn patch_kernelbase(args: Option<&[u16]>, kernelbase_ptr: *mut u8) {
    if let Some(args) = args {
        let peb = (*teb()).ProcessEnvironmentBlock;
        // This buffer pointer should match the cached UNICODE_STRING in kernelbase
        let buffer = (*(*peb).ProcessParameters).CommandLine.Buffer;

        // Search this pointer in kernel32's .data section
        if let Some(kernelbase_data) = get_module_section(kernelbase_ptr, b".data") {
            let ptr = kernelbase_data.as_mut_ptr();
            let len = kernelbase_data.len() / 2;
            // Do not have two mutable references to the same memory range

            let data_as_wordsize = core::slice::from_raw_parts(ptr as *const usize, len);
            if let Some(found) = data_as_wordsize
                .iter()
                .position(|ptr| *ptr == buffer as usize)
            {
                // We originally found this while scanning usize-sized data, so we have to translate
                // this to a byte index
                let found_buffer_byte_pos = found * core::mem::size_of::<usize>();
                // Get the start of the unicode string
                let unicode_str_start =
                    found_buffer_byte_pos - core::mem::offset_of!(UNICODE_STRING, Buffer);
                let unicode_str = core::mem::transmute::<_, &mut UNICODE_STRING>(
                    ptr.offset(unicode_str_start as isize),
                );

                let args_byte_len = args.len() * core::mem::size_of::<u16>();
                unicode_str.Buffer = args.as_ptr() as *mut _;
                unicode_str.Length = args_byte_len as u16;
                unicode_str.MaximumLength = args_byte_len as u16;
            }
        }
    }
}
```

### Preventing Host Application Crashes

I thought a great idea to prevent the "host application" (i.e. the application whose address space we're hijacking) from crashing by suspending all of its threads. I was surprised to learn that not only was this fairly easy to do on Windows, it was _even_ easier to accidentally do this from a non-admin session for all other Medium-IL processes!

[![Tweet by @landaire with text, "it has been 0 minutes since I last accidentally suspended all medium-IL threads on my system"](/img/pe-loader/thread_suspension.png)](/img/pe-loader/thread_suspension.png)

Here is the code I eventually came up with:

```rust
pub unsafe fn suspend_threads(kernel32_ptr: PVOID, kernelbase_ptr: PVOID) {
    // kernel32legacy.dll on xbox
    let CreateToolhelp32Snapshot = fetch_create_tool_help32(kernel32_ptr);
    let Thread32Next = fetch_thread_32_next(kernel32_ptr);
    let Thread32First = fetch_thread_32_first(kernel32_ptr);

    // kernelbase.dll on xbox
    let GetCurrentThreadId = fetch_get_current_thread_id(kernelbase_ptr);
    let GetCurrentProcessId = fetch_get_current_process_id(kernelbase_ptr);
    let OpenThread = fetch_open_thread(kernelbase_ptr);
    let SuspendThread = fetch_suspend_thread(kernelbase_ptr);
    let CloseHandle = fetch_close_handle(kernelbase_ptr);

    let pid = GetCurrentProcessId();
    // Suspend all other threads except this one
    let h = CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD, pid);
    let current_thread = GetCurrentThreadId();
    let mut te: THREADENTRY32 = core::mem::zeroed();
    te.dwSize = core::mem::size_of_val(&te) as u32;
    if Thread32First(h, &mut te as *mut _) != 0 {
        loop {
            if te.dwSize as usize
                >= offset_of!(THREADENTRY32, th32OwnerProcessID)
                    + core::mem::size_of_val(&te.th32OwnerProcessID)
            {
                if te.th32OwnerProcessID == pid {
                    if current_thread != te.th32ThreadID {
                        let thread_handle =
                            OpenThread(THREAD_SUSPEND_RESUME, false, te.th32ThreadID);
                        SuspendThread(thread_handle);
                    }
                }
            }
            if Thread32Next(h, &mut te as *mut _) == 0 {
                break;
            }
            te.dwSize = core::mem::size_of_val(&te) as u32;
        }
    }

    CloseHandle(h);
}
```

### Subtle Differences on Xbox

I'll top this section off with some random subtle differences I noticed about Xbox:

- The process environment block (PEB) is marked as readonly, but was not on my PC (latest Windows 11 as of writing this post). Pretty simple fix, just mark the PEB as writable before changing it... but stil interesting.
- `kernel32.dll` does not exist. Instead, some of its functionality is split between `kernelbase.dll` (which exists on Windows of course) and `kernel32legacy.dll`. If you would have found the function in _only_ `kernel32.dll` before, it probably now exists in `kernel32legacy.dll`.
