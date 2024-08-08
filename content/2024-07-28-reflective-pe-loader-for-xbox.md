+++
title = "Writing a Reflective PE Loader in 2024 for the Xbox"
description = "Adventures in reinventing the wheel. Also: I hate thread-local storage"
summary = "Adventures in reinventing the wheel. Also: I hate thread-local storage"
template = "toc_page.html"
toc = true
date = "2024-07-28"

[extra]
#image = "/img/wows-obfuscation/header.png"
#image_width =  250
#image_height = 250
hidden = true
pretext = """
Adventures in reinventing the wheel
"""
+++

*Absolutely nothing new is presented in this blog post, but you might learn something like I did. Full source code can be found [on our GitHub](https://github.com/exploits-forsale/solstice).*

Emma ([@carrot_c4k3](https://twitter.com/carrot_c4k3)) is a good friend of mine. We met about 17 years ago in the Xbox 360 scene and have remained friends ever since.

Emma recently participated in pwn2own in the Windows LPE category and ended up using a great bug for LPE. The bug far exceeded the category though: this vulnerability was also a _sandbox escape_, i.e. it's in an NT syscall which is reachable from the UWP sandbox. A couple months ago she got a wild idea: why not try to port the exploit over to the Xbox One?

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

All of the operating systems for each VM are a very slimmed down version of Windows based on Windows Core OS (WCOS) and the Hyper-V architecture is mostly what you'd encounter on a normal PC but with some additional Xbox-specific VSPs/functionality.

Missing from the above diagram is the _security processor_ (SP). The Xbox One's security processor should be the only thing on the Xbox which can reveal a title's plaintext on Xbox One. (*Random fact: Microsoft's [Pluton Processor](https://learn.microsoft.com/en-us/windows/security/hardware-security/pluton/microsoft-pluton-security-processor) is based on learnings from the Xbox One's security processor*)

The core idea behind all of this is to **make piracy extremely difficult**, if not impossible without breaking the SP. If you _do_ hack the Xbox One, you can't do it online trivially because the SP will attest that the console's state is something unexpected.

## OK, How Does This Relate to the PE loader?

Unrelated to her pwn2own entry, Emma found a vulnerability/feature in an application on the Xbox One marketplace called _GameScript_, which is an ImGui UI for messing with the [Ape programming language](https://github.com/kgabis/ape). Through this vulnerability Emma was able to read/write arbitrary memory and run shellcode. So we have arbitrary code execution in SystemOS, but now the problem: writing shellcode is a pain, so how can we run arbitrary *executables* easily?

We have the ability to read/write arbitrary memory and change page permissions, so Emma asked if I would write a PE loader. It would be required to simplify the development pipeline while she worked on porting her exploit over and it'll be useful for homebrew later on too. Easy enough right?

**Wrong.**

## Reinventing the Wheel

The specific technique of user-mode portable executable (PE/.exe file) loading is referred as "Reflective PE Loading" which to me is some #redteam term I'd never heard before embarking on this project. This name is meaningless in my opinion, but a "reflective PE loader" is simply some user-mode code that can load a PE without going through the normal `LoadLibrary()` / `CreateProcess()` routines and executes the PE's entrypoint.

Avoiding `LoadLibrary()` and `CreateProcess()` is very important for us since these APIs will go through code integrity (CI)... and any code we write will not be properly signed. I took a look at the work involved and decided I wanted to write my own loader for a few reasons:

1. I despise dealing with C/C++ build systems.

2. Since I'm targeting _Xbox Windows_ and not _desktop Windows_, I might encounter some problems and I know how to debug my own code better than someone else's.

3. On the Xbox we're going to _have to_ use a PE loader for running any executable until we eventually break code integrity. So we better know how it works and be able to load complex applications.

4. I don't give a shit about EDR evasion or any #redteam stuff like that.

5. It seemed simple enough at the time to just rewrite it in Rust, so I did.

For my project's base I combined two open-source Rust projects:

- [b1tg/rust-windows-shellcode](https://github.com/b1tg/rust-windows-shellcode) which provided a great template for writing and building Windows shellcode in Rust.
- [Thoxy67/rspe](https://github.com/Thoxy67/rspe) which provides a basic reflective loader.

rspe already got me most of the way there, but with a few caveats:

- It needed some cleanup (e.g. lots of unnecessary copies)
- It did not support loading imports by ordinal
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

As you might have noticed, we're not linking against any libraries and calling those imports directly. Instead we're using indirect calls to functions whose addresses we manually resolved at runtime.

## The Easy Parts

[Although it's been talked about before](https://github.com/BenjaminSoelberg/ReflectivePELoader?tab=readme-ov-file), I'll give a brief overview of how a basic loader works:

1. Parse the PE headers and `VirtualAlloc()` some memory for the "cloned" PE with all the fixups applied. You'll try to `VirtualAlloc()` at the PE's preferred load address, but if you don't get it fall back to a random address. This is your _load address_. From here you calculate the delta between the preferred and actual load address and this will be used for fixing relocations.

2. [Iterate each PE section and copy it over to the newly `VirtualAlloc`'d region.](https://github.com/exploits-forsale/solstice/blob/6c47b5a0cd155d629845412974e7580fa9dff840/crates/solstice_loader/src/pelib.rs#L211-L254) The virtual addresses here are _relative_ virtual addresses, so you just take each section's VirtualAddress, add it to the load address, and copy the section from its old location to the new address.

3. [Fix section permissions.](https://github.com/exploits-forsale/solstice/blob/6c47b5a0cd155d629845412974e7580fa9dff840/crates/solstice_loader/src/pelib.rs#L256-L321) For each section, look at its `Characteristics` field and determine the correct permissions. `VirtualProtect()` the section according to the permissions.

4. [Fix imports.](https://github.com/exploits-forsale/solstice/blob/6c47b5a0cd155d629845412974e7580fa9dff840/crates/solstice_loader/src/pelib.rs#L425-L545). For each import in the import table (`IMAGE_DIRECTORY_ENTRY_IMPORT`), ensure the imported DLL is loaded. Then use the loaded DLL's handle with `GetProcAddress()` to get the address of the function being imported. For each import in the table, write the real address in the import's thunk. Instead of `GetProcAddress()` could also parse the module's exports and match things up, but I took the lazy way.

5. [Fix relocations.](https://github.com/exploits-forsale/solstice/blob/6c47b5a0cd155d629845412974e7580fa9dff840/crates/solstice_loader/src/pelib.rs#L323-L398) This basically involves walking the `IMAGE_DIRECTORY_ENTRY_BASERELOC` directory and fixing each `IMAGE_BASE_RELOCATION` such that you add the delta calculated in step 1 to the relocation's `VirtualAddress` field. There's some nuance here where you need to only modify certain bits, etc. etc. but this is the basic idea.

6. [Call the module's entrypoints.](https://github.com/exploits-forsale/solstice/blob/main/crates/solstice_loader/src/lib.rs#L343-L347)

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

unsafe fn execute_tls_callback(baseptr: *const c_void, entrypoint: *const c_void) {
    let func: ImageTlsCallbackFn = core::mem::transmute(entrypoint);
    func(baseptr, DLL_THREAD_ATTACH, ptr::null_mut());
}
```

Executing the image entrypoint is pretty similar:

```rust
let entrypoint = (baseptr as usize
    + (*(ntheader as *const IMAGE_NT_HEADERS64))
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

There were some parts that really kicked my ass in figuring out, but in my opinion were very important for what I wanted in the PE loader.

1. The exploit / PE loader must not cause the hijacked application to become unreliable. I don't want to be debugging crashes in some of the existing threads that broke simply because we're hijacking the address space.

2. We must be able to run complex applications. Since we're using this technique to bypass code integrity, this will be our main method of running arbitrary applications.

3. The application shouldn't _know_ it's been reflectively loaded, or care.

### Thread-Local Storage

Related to #2, the absolute biggest challenge I faced was with applications that use thread-local storage. Having done all of my development in Rust, my test program that I was loading was also written in Rust.

I kept encountering `int 29` instructions (`RtlFailFast(code)`) that crashed the application when I'd execute its entrypoint. This was **extremely** painful to debug, but eventually I figured out that I was failing after fetching data from thread-local storage:

[![Screenshot of assembly instructions from a Rust "hello world" application loading data from thread-local storage in IDA pro](/img/pe-loader/tls-thread-set-current.png)](/img/pe-loader/tls-thread-set-current.png)

[![Screenshot of assembly instructions from a Rust "hello world" application executing an `int 29` instruction in IDA pro](/img/pe-loader/int-29.png)](/img/pe-loader/int-29.png)

I was kind of confused because I didn't expect my application to use thread-local storage, but apparently even the most basic "hello world" Rust program uses TLS:

[![Screenshot of a Rust "hello world" application loaded into the "PE Bear" program, showing its TLS directory](/img/pe-loader/pe-bear-tls.png)](/img/pe-loader/pe-bear-tls.png)

It turns out that this is related to Rust's thread initialization code that sets some thread-locals for the current thread and thread ID: [https://github.com/rust-lang/rust/blob/2e630267b2bce50af3258ce4817e377fa09c145b/library/std/src/thread/mod.rs#L694](https://github.com/rust-lang/rust/blob/2e630267b2bce50af3258ce4817e377fa09c145b/library/std/src/thread/mod.rs#L694)

So I came to realize that my original idea for how I was handling thread-local storage was completely flawed. Originally I was _allocating_ new memory for my module's TLS, but didn't even realize it had some default state associated with it that I had to copy over. Simple fix right?

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

This code *worked*, but it didn't work for long. I obviously had no idea how thread-local storage worked, and soon discovered that in a multi-threaded application I was _again_ getting similar crashes because the TLS data was bad. Through much pain and debugging I ended up learning:

- Changing the thread-local storage for your current thread is obviously not enough. New threads that spawn won't have the modifications I did above, so they'll have "default" TLS without my module included since the changes I did above are only reflected for the current thread. Duh.

- TLS is allocated in slots for the current thread and each slot is a pointer to the TLS data.

- Windows keeps a cache of TLS directories for each loaded module, which means you can't just pave over the hijacked module's TLS data with your new TLS data and things will "just work". You'll have to update the cache.

### Fixing TLS Data

In the above section I mentioned that Windows keeps a cache of TLS directories for each loaded module, and I think this is a critical reason why the reflective PE loaders I sampled didn't bother with TLS data ([only one loader sampled seemed to support TLS data](https://github.com/DarthTon/Blackbone/blob/5ede6ce50cd8ad34178bfa6cae05768ff6b3859b/src/BlackBone/ManualMap/Native/NtLoader.cpp#L153)).

I really only discovered this by painfully debugging and figuring out the application only crashed when spawning new threads, that the crashes were relating to data in TLS, and figuring that something must be wrong with the TLS data.

It finally clicked when I noticed that the `ThreadLocalStoragePointer` for the crashing thread's TEB didn't match the spawning thread's...

[![!teb command in WinDbg](/img/pe-loader/teb-command.png)](/img/pe-loader/teb-command.png)

[![Clicking the TEB pointer in WinDbg's !teb output](/img/pe-loader/thread-local-storage.png)](/img/pe-loader/thread-local-storage.png)

This is super obvious in hindsight! Each thread's TLS has to be unique, but I don't know... I thought the `ThreadLocalStoragePointer` was a pointer to the _default state_ TLS and the per-thread slots were in the TEB's `TlsSlots` field?

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

#### Not Great Approaches to Fixing TLS Data

Method 1 has a big problem: if the program you're loading requires TLS, you must inject into a program with TLS. Otherwise you'll be replacing a random DLL's TLS data. Unless you're very careful that could be a Window component you need to use.

{% collapse(preview="Method 1 -- List Patching (Least Worst)") %}

I popped `ntdll.dll` into IDA to see what functions were using this `LdrpTlsList` to see if maybe there was some other way I could grab the list's address.

[![IDA Pro window showing functions using LdrpTlsList](/img/pe-loader/LdrpFindTlsEntry.png)](/img/pe-loader/LdrpFindTlsEntry.png)

I found that in Windows (but not ReactOS) is a function, "LdrpFindTlsList", which will return a `PTLS_ENTRY` (the actual name of the Windows data structure for ReactOS's `LDRP_TLS_DATA`) given a `PLDR_DATA_TABLE_ENTRY`. [Ken Johnson even conviently provided the source code on his blog](http://www.nynaeve.net/Code/VistaImplicitTls.cpp).

So now the only missing link: finding the `PLDR_DATA_TABLE_ENTRY`. I'm kind of doing a "draw the rest of the owl" moment, but figuring this out was fairly straightforward by poking through the `!peb` command in WinDbg.

The complete code:

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

    // This stuff here is completely unnecessary, but I did it anyways as a "just in case"
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

{% end %}

Method 2's problems:

1. The loader may be using a TLS bitmap, which isn't covered by either of the above cases.
2. Running threads are unaffected.
3. There may be persistent data in the PEB that we aren't updating.
4. I had bizzaro crashes that I didn't even bother investigating.

{% collapse(preview="Method 2 -- Allocate a New TLS Entry (Terrible)") %}
I later realized that the function `LdrpAllocateTlsEntry` does _almost_ all of the above work for me for free and doesn't check if the current module already has a TLS slot allocated. Using this function to allocate the TLS slot would also allow me to inject programs that use TLS data into programs that do not have any themselves!

I could therefore rewrite my loader to do the following instead:

```rust
const LDRP_ALLOCATE_TLS_ENTRY_SIGNATURE_BYTES: [u8; 8] = [
    0x4C, 0x89, 0x4C, 0x24, 0x20, 0x4C, 0x89, 0x44, 0x24, 0x18, 0x48, 0x89, 0x54, 0x24, 0x10, 0x53,
    0x56, 0x57, 0x41, 0x56, 0x41, 0x57, 0x48, 0x83, 0xEC, 0x30, 0x49, 0x8B,
];

// Signature taken from http://www.nynaeve.net/Code/VistaImplicitTls.cpp
type LdrpAllocateTlsEntryFn = unsafe extern "system" fn(
    image_tls_dir: *const IMAGE_TLS_DIRECTORY64,
    entry: *mut LDR_DATA_TABLE_ENTRY,
    tls_index: &mut u32,
    allocated_bitmap: *mut c_void,
    tls_entry: *mut *const c_void,
) -> NTSTATUS;

/// Patches the module list to change the old image name to the new image name.
///
/// This is useful to ensure that a program that depends on `GetModuleHandle*`
/// doesn't fail simply because its module is not found
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
    let mut tls_index = 0;
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
                    for window in ntdll_text.windows(LDRP_ALLOCATE_TLS_ENTRY_SIGNATURE_BYTES.len())
                    {
                        if window == LDRP_ALLOCATE_TLS_ENTRY_SIGNATURE_BYTES {
                            // Get this window's pointer -- it should be to the start of the function
                            let mut ptr = window.as_ptr();
                            let LdrpAllocateTlsEntry: LdrpAllocateTlsEntryFn =
                                core::mem::transmute(ptr);

                            let mut tls_entry: *const c_void = core::ptr::null();
                            LdrpAllocateTlsEntry(
                                this_tls_data,                   // ImageName
                                module_info,                     // Entry
                                &mut tls_index,                  // TlsIndex
                                core::ptr::null_mut(),           // AllocatedBitmap
                                &mut tls_entry as *mut *const _, // TlsEntry
                            );
                        }
                    }
                }
            }
            break;
        }
        next = (*next).Flink;
    }

    tls_index
}
```
{% end %}

#### The Good Method

Remember how I said [only one loader sampled seemed to support TLS data](https://github.com/DarthTon/Blackbone/blob/5ede6ce50cd8ad34178bfa6cae05768ff6b3859b/src/BlackBone/ManualMap/Native/NtLoader.cpp#L153)? This happens to be the same approach they took.

I looked at who calls `LdrpAllocateTlsEntry` (method #2) and a private function `LdrpHandleTlsData`, which is called when a new module is loaded, has no sanity checks on whether or not the module's TLS data has already been handled. Which is awesome, and actually makes sense!

Why sanity check if this function is only ever called once during real loader scenarios?

We can abuse this by performing the following operations:

1. Update the hijacked module's `LDR_DATA_TABLE_ENTRY` to point to our new module's base address.
2. Release the hijacked module's TLS data (`LdrpReleaseTlsEntry`)
3. Call `LdrpHandleTlsData` with the hijacked module to force the new TLS data to be loaded.

This also solves all of the problems we had with both prior methods!

- We can inject into any process and not just processes that have TLS data
- According to the [Ken Johnson code](http://www.nynaeve.net/Code/VistaImplicitTls.cpp) this function updates the TLS info in the PEB (or maybe some kernel data?)
- And according to the Ken Johnson code updates other threads
- Is less code than _both_ other solutions
- Doesn't require me to manually update the new module's TLS index

```rust
const LDRP_RELEASE_TLS_ENTRY_SIGNATURE_BYTES: [u8; 7] = [0x83, 0xE1, 0x07, 0x48, 0xC1, 0xEA, 0x03];

const LDRP_HANDLE_TLS_DATA_SIGNATURE_BYTES: [u8; 9] =
    [0xBA, 0x23, 0x00, 0x00, 0x00, 0x48, 0x83, 0xC9, 0xFF];

type LdrpReleaseTlsEntryFn =
    unsafe extern "system" fn(entry: *mut LDR_DATA_TABLE_ENTRY, unk: *mut c_void) -> NTSTATUS;

type LdrpHandleTlsDataFn = unsafe extern "system" fn(entry: *mut LDR_DATA_TABLE_ENTRY);

/// Patches the module list to change the old image name to the new image name.
///
/// This is useful to ensure that a program that depends on `GetModuleHandle*`
/// doesn't fail simply because its module is not found
pub unsafe fn patch_ldr_data(
    new_base_address: *mut c_void,
    module_size: usize,
    get_module_handle_fn: GetModuleHandleAFn,
    this_tls_data: *const IMAGE_TLS_DIRECTORY64,
    entrypoint: *const c_void,
) {
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
                    // Get the TLS entry for the current module and remove it from the list
                    for window in ntdll_text.windows(LDRP_RELEASE_TLS_ENTRY_SIGNATURE_BYTES.len()) {
                        if window == LDRP_RELEASE_TLS_ENTRY_SIGNATURE_BYTES {
                            // Get this window's pointer. It will land us in the middle of this function though
                            let mut ptr = window.as_ptr();
                            // Walk backwards until we find the prologue. Pray this function retains padding
                            loop {
                                if *ptr.offset(-1) == 0xcc && *ptr.offset(-2) == 0xcc {
                                    break;
                                }
                                ptr = ptr.offset(-1);
                            }

                            // Get this window's pointer and move backwards to find the start of the fn
                            #[allow(non_snake_case)]
                            let LdrpReleaseTlsEntry: LdrpReleaseTlsEntryFn =
                                core::mem::transmute(ptr);

                            LdrpReleaseTlsEntry(module_info, core::ptr::null_mut());

                            break;
                        }
                    }

                    for window in ntdll_text.windows(LDRP_HANDLE_TLS_DATA_SIGNATURE_BYTES.len()) {
                        if window == LDRP_HANDLE_TLS_DATA_SIGNATURE_BYTES {
                            // Get this window's pointer. It will land us in the middle of this function though
                            let mut ptr = window.as_ptr();
                            // Walk backwards until we find the prologue. Pray this function retains padding
                            loop {
                                if *ptr.offset(-1) == 0xcc && *ptr.offset(-2) == 0xcc {
                                    break;
                                }
                                ptr = ptr.offset(-1);
                            }

                            #[allow(non_snake_case)]
                            let LdrpHandleTlsData: LdrpHandleTlsDataFn = core::mem::transmute(ptr);

                            LdrpHandleTlsData(module_info);

                            break;
                        }
                    }
                }
            }
            break;
        }
        next = (*next).Flink;
    }
}
```

### Patching Command-Line Args and Image Name

This has been done by other PE loaders, but I wanted to call this out as well: while the PEB contains the image name and process arugments, so does `kernelbase.dll`! Why? For `GetCommandLineW` and `GetCommandLineA` of course.

This one wasn't _too_ bad to patch so long as you want to rely on the fact that the `UNICODE_STRING` structure for the PEB and in `kernelbase.dll` share the same backing buffer (i.e. the latter is a shallow copy of the former). That also doesn't account for the `ANSI_STRING` variant... but 🤷‍♂️

tl;dr of the following code: we scan the global memory of `kernelbase.dll` looking for the previously mentioned `UNICODE_STRING` buffer pointer we obtained from the PEB then, once found, update its pointer and length to match our new pointer and length.

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

### Preventing Hijacked Application Crashes

I thought a great idea to prevent the hijacked application from crashing by suspending all of its threads. I was surprised to learn that not only was this fairly easy to do on Windows, it was _even_ easier to accidentally do this from a non-admin session for all other Medium-IL processes!

[![Tweet by @landaire with text, "it has been 0 minutes since I last accidentally suspended all medium-IL threads on my system"](/img/pe-loader/thread_suspension.png)](/img/pe-loader/thread_suspension.png)

_Yeah, don't call `CreateToolhelp32Snapshot()` incorrectly_.

The Windows examples were actually fairly straightforward but on Xbox the code crashed. And that's because the `kernel32_ptr` here actually needs to be a pointer to `kernel32legacy.dll` since on Xbox `kernel32.dll` doesn't exist.

That took me a while to figure out and hunt down and double-check where the functions got relocated to.

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

## fin

This was a fun exercise that taught me a lot about how Windows binaries are loaded. I'd like to thank carrot_c4k3, tuxuser, and 0e9ca321209eca529d6988c276e4e4ed for their help/support.

With this work, we're now able to do cool things on Xbox :)

[![Collateral Damage Executed Achievement](/img/pe-loader/collat_achievement.webp)](/img/pe-loader/collat_achievement.webp)
