
+++
title = "One Weird Trick to Improve Bug Finding With ASAN"
summary = ""
template = "toc_page.html"
toc = true

[extra]
image = "/img/asan/asan_header2.png"
image_width = 1000
image_height = 500
pretext = """
*...ok, three weird tricks*
"""
+++

## ASAN Primer

*If you're already an ASAN expert, feel free to skip to the next section.*

[AddressSanitizer](https://clang.llvm.org/docs/AddressSanitizer.html) (ASAN) is an extremely useful tool in software testing, debugging, and security testing for finding memory safety issues in native applications. It's extremely straightforward to use on most platforms -- all you need to do is pass `-fsanitize=address` to clang/gcc and run the application.

As your application runs it builds metadata about its memory state into what's called a *shadow memory*. The shadow memory is essentiallly a compressed representation of the application's address space and is used to look up memory ranges that are considered addressable. Memory ranges that are not addressable will be referred to as "poisoned memory".

When ASAN detects a memory safety issue it will print a report to the console and stop the application. The first bit of the report is as follows:

```
=================================================================
==1==ERROR: AddressSanitizer: container-overflow on address 0x602000000010 at pc 0x560696424930 bp 0x7ffce1e0f150 sp 0x7ffce1e0f148
WRITE of size 4 at 0x602000000010 thread T0
    #0 0x56069642492f in main /app/example.cpp:14:15
    #1 0x7fecaaf9a082 in __libc_start_main (/lib/x86_64-linux-gnu/libc.so.6+0x24082) (BuildId: 1878e6b475720c7c51969e69ab2d276fae6d1dee)
    #2 0x56069636335d in _start (/app/output.s+0x2135d)

0x602000000010 is located 0 bytes inside of 12-byte region [0x602000000010,0x60200000001c)
allocated by thread T0 here:
    #0 0x56069642211d in operator new(unsigned long) /root/llvm-project/compiler-rt/lib/asan/asan_new_delete.cpp:95:3
    #1 0x560696427824 in void* std::__1::__libcpp_operator_new[abi:v15000]<unsigned long>(unsigned long) /opt/compiler-explorer/clang-15.0.0/bin/../include/c++/v1/new:246:10
    #2 0x560696427808 in std::__1::__libcpp_allocate[abi:v15000](unsigned long, unsigned long) /opt/compiler-explorer/clang-15.0.0/bin/../include/c++/v1/new:272:10
    #3 0x5606964277a9 in std::__1::allocator<Foo>::allocate[abi:v15000](unsigned long) /opt/compiler-explorer/clang-15.0.0/bin/../include/c++/v1/__memory/allocator.h:112:38
    #4 0x5606964275e0 in std::__1::__allocation_result<std::__1::allocator_traits<std::__1::allocator<Foo>>::pointer> std::__1::__allocate_at_least[abi:v15000]<std::__1::allocator<Foo>>(std::__1::allocator<Foo>&, unsigned long) /opt/compiler-explorer/clang-15.0.0/bin/../include/c++/v1/__memory/allocate_at_least.h:54:19
    #5 0x560696426479 in std::__1::__split_buffer<Foo, std::__1::allocator<Foo>&>::__split_buffer(unsigned long, unsigned long, std::__1::allocator<Foo>&) /opt/compiler-explorer/clang-15.0.0/bin/../include/c++/v1/__split_buffer:316:29
    #6 0x560696425927 in void std::__1::vector<Foo, std::__1::allocator<Foo>>::__push_back_slow_path<Foo>(Foo&&) /opt/compiler-explorer/clang-15.0.0/bin/../include/c++/v1/vector:1535:49
    #7 0x560696424d3b in std::__1::vector<Foo, std::__1::allocator<Foo>>::push_back[abi:v15000](Foo&&) /opt/compiler-explorer/clang-15.0.0/bin/../include/c++/v1/vector:1567:9
    #8 0x5606964248d1 in main /app/example.cpp:11:10
    #9 0x7fecaaf9a082 in __libc_start_main (/lib/x86_64-linux-gnu/libc.so.6+0x24082) (BuildId: 1878e6b475720c7c51969e69ab2d276fae6d1dee)
```

It tells us there's a *container-overflow*, what address the container overflow occurred at, the call stack of where the overflow occurred, and finally where the memory we're faulting on was originally allocated.

The next bit of the report is the shadow memory that was mentioned above:

{{ resize_image(path="/img/asan/asan_error_with_arrow.png", width=500, height=800, op="fit") }}

The big arrow here is pointing into the shadow memory at `[06]`, which according to the legend at the bottom of the screenshot tells us there are six addressable bytes followed by a *global redzone* (represented by the red 0xf9 in the shadow bytes).

### Runtime Instrumentation

ASAN builds its shadow memory at runtime with the help of its runtime library, `libclang_rt.asan_{target_platform}_dynamic.dylib`. The runtime library provides some of the following:

- Memory management hooks for `malloc()`, `free()`, `operator new()`, etc. Whenever a memory allocation/free occurs ASAN will update its shadow memory
- Functions for checking if memory is addressable or poisoned.
- Hooks for some common memory manipulation functions (`strncpy`, `strcpy`, `memcpy`, `memcmp`, etc.).

For checking if memory is addressable, ASAN's runtime provides some simple APIs that are used by its compiler instrumentation such as:

- `__asan_load1`
- `__asan_store1`
- `__asan_load2`
- `__asan_store2`
- `__asan_load8`
- `__asan_store8`
- ...
- `__asan_loadN`

*Note: This is certainly not a definitive list of APIs, but are relatively common*.

These all essentially do the same thing under the hood:

```c
extern "C" NOINLINE INTERFACE_ATTRIBUTE void __asan_exp_loadN(uptr addr, uptr size,
u32 exp) {

    if (__asan_region_is_poisoned(addr, size)) {
        GET_CALLER_PC_BP_SP;
        ReportGenericError(pc, bp, sp, addr, false, size, exp, true, false);
    }

}
```

They take an address and size, check if memory in that range is poisoned, and reports a generic error if it is.

### Compiler Instrumentation

The compiler instrumentation is primarily used for poisoning stack memory and inserting calls into the runtime library for "interesting" loads/stores. Of course, not *every* load/store will be instrumented by ASAN as that'd be a bit too heavy weight and a lot of things can be determined to be "safe" statically in the compiler.

I'm not a compiler expert and truthfully don't care to dive into the source code at this time to figure out how ASAN determines what an "interesting" load/store is. With that said, when one is encountered ASAN's compiler pass will insert calls to the `__asan_{load,store}{size}` runtime functions to check the operation.

## You're Probably Missing Out-of-Bounds Accesses

With the crash course on ASAN out of the way, we can dive in to the main point of this blog post: you're probably missing OOBR/W in your applications if you're using C++/Rust/whatever language containers.


### The Problem With Vectors

Here is some example code that should raise an out-of-bounds access violation:

```cpp
#include <vector>
#include <stdio.h>
#include <string.h>

int main() {
    // Allocate a vector to store some data generated by our fuzzer
    std::vector<char> fuzzed;
    // Fuzzer pushes 5 bytes to the vector
    fuzzed.push_back(0x41);
    fuzzed.push_back(0x42);
    fuzzed.push_back(0x43);
    fuzzed.push_back(0x44);
    fuzzed.push_back(0x45);

    // Copy 8 bytes from the vector to a test buffer
    char test[8] = {0};
    memcpy(&test, fuzzed.data(), sizeof(test));
    for (size_t i = 0; i < sizeof(test); i++) {
        printf("%02X", test[i]);
    }

    printf("\nsize(%zu), capacity(%zu)\n", fuzzed.size(), fuzzed.capacity());

    return 0;
}
```

We have a vector with 5 bytes that we then try to copy 8 bytes from. Pretty standard out-of-bounds read. When we run this with ASAN however...

```
Program returned: 0
Program stdout

4142434445FFFFFFBEFFFFFFBEFFFFFFBE
size(5), capacity(8)
```

*[https://godbolt.org/z/cecf6Pjz8](https://godbolt.org/z/cecf6Pjz8)*

No crash! You might notice something interesting in the last line of the output though: the size of the vector is 5, but its capacity is *8*.

Some readers probably know that when you `push_back()` or insert data into a `vector` that's at its capacity, it reallocates the buffer to be *double* its current size, copies the data to the new buffer, and frees the old one (or just does a `realloc()`). As a vector starts to grow from 0 elements up to N, its growth looks like the following:

![Vector growth strategy](/img/asan/vector_growth.png)

*[Source](https://i.stack.imgur.com/w5VP7.png)*

This is very problematic for us. We're not catching an out-of-bounds access because of some implementation detail. All ASAN knows is that the application requested a buffer with 8 bytes -- it doesn't know that in our case 3 of those bytes are unused memory that aren't safe for us to use yet.

In the general case, any memory accesses in the range from `[vector.data() + vector.size(), vector.data() + vector.capacity()]` won't be detected as an out-of-bounds access!

### The Problem With Strings

Here's an example that's basically the same as the vector example above -- except, we're now constructing an `std::string` with a static C string.

```cpp
#include <stdio.h>
#include <string>
#include <string.h>

int main() {
    std::string test("four");
    char temp[10] = {0};
    memcpy(&temp, test.data(), sizeof(temp));

    for (size_t i = 0; i < sizeof(temp); i++) {
        printf("%02X", temp[i]);
    }

    printf("\nsize(%lu), capacity(%lu)\n", test.size(), test.capacity());

    return 0;
}

```

Again, this doesn't trigger a crash:

```
Program returned: 0
Program stdout

666F7572000000000000
size(4), capacity(15)
```

*[https://godbolt.org/z/hdjK1WoKo](https://godbolt.org/z/hdjK1WoKo)*


So the four-character string actually has a total capacity of 15, i.e. the `std::string` has over-allocated memory. If you tried initializing an `std::vector` with an explicit initializer list it would allocate only the exact number of elements needed... why are strings different?

Let's take a look at LLVM's libc++ `string` code (simplified version will follow):

```cpp
#ifdef _LIBCPP_BIG_ENDIAN
    static const size_type __short_mask = 0x01;
    static const size_type __long_mask  = 0x1ul;
#else  // _LIBCPP_BIG_ENDIAN
    static const size_type __short_mask = 0x80;
    static const size_type __long_mask  = ~(size_type(~0) >> 1);
#endif // _LIBCPP_BIG_ENDIAN

    enum {__min_cap = (sizeof(__long) - 1)/sizeof(value_type) > 2 ?
                      (sizeof(__long) - 1)/sizeof(value_type) : 2};

    struct __short
    {
        value_type __data_[__min_cap];
        struct
            : __padding<value_type>
        {
            unsigned char __size_;
        };
    };

#else

    struct __long
    {
        size_type __cap_;
        size_type __size_;
        pointer   __data_;
    };

#ifdef _LIBCPP_BIG_ENDIAN
    static const size_type __short_mask = 0x80;
    static const size_type __long_mask  = ~(size_type(~0) >> 1);
#else  // _LIBCPP_BIG_ENDIAN
    static const size_type __short_mask = 0x01;
    static const size_type __long_mask  = 0x1ul;
#endif // _LIBCPP_BIG_ENDIAN

    enum {__min_cap = (sizeof(__long) - 1)/sizeof(value_type) > 2 ?
                      (sizeof(__long) - 1)/sizeof(value_type) : 2};

    struct __short
    {
        union
        {
            unsigned char __size_;
            value_type __lx;
        };
        value_type __data_[__min_cap];
    };

#endif // _LIBCPP_ABI_ALTERNATE_STRING_LAYOUT

    union __ulx{__long __lx; __short __lxx;};

    enum {__n_words = sizeof(__ulx) / sizeof(size_type)};

    struct __raw
    {
        size_type __words[__n_words];
    };

    struct __rep
    {
        union
        {
            __long  __l;
            __short __s;
            __raw   __r;
        };
    };

    __compressed_pair<__rep, allocator_type> __r_;
```

[GitHub link.](https://github.com/landaire/llvm-project/blob/f860d2e78cca40e2b8697a22a92efebfea409256/libcxx/include/string#L731-L803)

*Yuck*. This is not simple to understand, but we can see that there's some interesting inline buffer stuff going on with the `__short` struct at least. I've rewritten this code to be *definitely not* the same layout as an `std::string` but shows what's going on easier to understand:

```cpp
class string {
	char short_optimization[15];
	size_t len;
	size_t capacity;
	char *heap_longer_string;
}
```

`std::string` has an optimization for short strings that allows it to avoid a heap allocation. Unfortunately, this means that for small strings we won't detect small out-of-bounds reads (OOBR) similar to the `std::vector` problem. And similar to the `std::vector` problem, heap-allocated strings grow in a way that over-allocates memory to reduce the number of allocations every time you push more data to it.

## Fixes

### The "One Weird Trick"

This isn't really documented anywhere, but `std::vector` actually does have ASAN enlightenment to detect this exact problem we're talking about:

```cpp
    // The following functions are no-ops outside of AddressSanitizer mode.
    // We call annotatations only for the default Allocator because other allocators
    // may not meet the AddressSanitizer alignment constraints.
    // See the documentation for __sanitizer_annotate_contiguous_container for more details.
#ifndef _LIBCPP_HAS_NO_ASAN
    _LIBCPP_CONSTEXPR_SINCE_CXX20
    void __annotate_contiguous_container(const void *__beg, const void *__end,
                                         const void *__old_mid,
                                         const void *__new_mid) const
    {

      if (!__libcpp_is_constant_evaluated() && __beg && is_same<allocator_type, __default_allocator_type>::value)
        __sanitizer_annotate_contiguous_container(__beg, __end, __old_mid, __new_mid);
    }
#else
    _LIBCPP_CONSTEXPR_SINCE_CXX20 _LIBCPP_HIDE_FROM_ABI
    void __annotate_contiguous_container(const void*, const void*, const void*,
                                         const void*) const _NOEXCEPT {}
#endif
```

[GitHub Link](https://github.com/llvm/llvm-project/blob/b7a2ff296352acacdc413d6f3f912e50f90ebb31/libcxx/include/vector#L740-L750).

When the `_LIBCPP_HAS_NO_ASAN` preprocessor macro is not defined it has some logic for informing ASAN about the contiguous region of a vector as well as the contiguous region that's allocated but yet-unused. The preprocessor macro is only defined when:

```
#    if !__has_feature(address_sanitizer)
#      define _LIBCPP_HAS_NO_ASAN
#    endif
```
[GitHub Link](https://github.com/llvm/llvm-project/blob/7ca3444fba7344b375f147b77252adbf71f464e0/libcxx/include/__config#LL479-L481C11).

So why the hell aren't we getting this enlightenment? We never defined it ourselves.

I don't even remember why I tried this, but it seems you need to explicitly pass `-stdlib=libc++` and just like magic, it works. Our example for an `std::vector` will now detect the small OOBR with this flag:

```
=================================================================
==1==ERROR: AddressSanitizer: heap-buffer-overflow on address 0x602000000075 at pc 0x5640917cd227 bp 0x7ffe3ad2ee30 sp 0x7ffe3ad2e600
READ of size 8 at 0x602000000075 thread T0
    #0 0x5640917cd226 in __asan_memcpy /root/llvm-project/compiler-rt/lib/asan/asan_interceptors_memintrinsics.cpp:22:3
    #1 0x56409180aa25 in main /app/example.cpp:17:5
    #2 0x7f35ef2f1082 in __libc_start_main (/lib/x86_64-linux-gnu/libc.so.6+0x24082) (BuildId: 1878e6b475720c7c51969e69ab2d276fae6d1dee)
    #3 0x56409174935d in _start (/app/output.s+0x2135d)
```

*[https://godbolt.org/z/ao64GcT7f](https://godbolt.org/z/ao64GcT7f).*

There are some downsides to this:

- `std::vector` is the only container with this enlightenment. But it does automatically update the poisoned region whenever we insert, remove, or clear the elements which is very nice.
- Our `std::string` example still doesn't detect the OOBR with this compiler flag: [https://godbolt.org/z/3bj6nnGxG](https://godbolt.org/z/3bj6nnGxG).
- You may not want to enable this if you have modules you cannot compile with this flag that may share an `std::vector`. The module that's not enlightened would not poison memory correctly, leading to false-positives. There may be ABI compatability issues as well.

### Code-Level Fix

[Google's ASAN wiki](https://github.com/google/sanitizers/wiki/AddressSanitizerManualPoisoning) provides documentation for how to manually poison memory yourself using `ASAN_POISON_MEMORY_REGION(addr, size)` and `ASAN_UNPOISON_MEMORY_REGION(addr, size)`. We can use this as follows:

```cpp
#if __has_feature(address_sanitizer) || defined(__SANITIZE_ADDRESS__)
#include <sanitizer/asan_interface.h>
#endif

const uint8_t *extra_start = fuzzed.data() + fuzzed.size();
size_t extra_len = fuzzed.capacity() - fuzzed.size();


#if __has_feature(address_sanitizer) || defined(__SANITIZE_ADDRESS__)
ASAN_POISON_MEMORY_REGION(extra_start, extra_len);
#endif
```

Or if for some reason you don't want to pull in the ASAN interface you could just copy data to a vector with the appropriate pre-allocated size:


```cpp
std::vector<uint8_t> copied(fuzzed.size());
std::copy(
    fuzzed.begin(),
    fuzzed.end(),
    std::back_inserter(copied)
);
assert_eq(copied.capacity(), copied.size())
```

Copying data sucks, but do what works for you. *Note:* avoid using `std::vector::shrink_to_fit()`. Per [cppreference](https://en.cppreference.com/w/cpp/container/vector/shrink_to_fit), "It depends on the implementation whether the request is fulfilled."

## Other Tricks

While I have your attention I wanted to call out some other things you can do to improve your ability to find bugs.

### Failfast

If you have an abstraction that's intended to safely handle memory, why wait for your test or fuzzing harness to find the bug? For example, in my opinion a `span` implementation should never be given an invalid memory range. We can enforce this at its constructor by checking if the provided memory region is poisoned and trigger a controlled crash:

```cpp
#include <sanitizer/asan_interface.h>

template<typename T>
class span<T> {
    public:
    span(T *data, size_t count) {

        if (__asan_region_is_poisoned(static_cast<void*>(data), count * sizeof(T))) {
            assert(false);
        }

    }
}
```

### Sanitizer Recovery

Whenever you repro a bug with ASAN, try to remember to compile with `-fsanitize-recover=address`. This will essentially allow the application to recover and continue running when ASAN triggers a violation.

It may seem like a strange choice, but let's say you have a small out-of-bounds read that looks relatively boring. That bug may be hiding something much juicier that's trigger *only* when the OOBR occurs! `-fsanitize-recover=address` will allow the application to run until either a hard fault occurs or the application exits, but will still print any ASAN violation that occurs along the way.

## Closing Thoughts

ASAN is a very powerful tool, but has limitations on what it can provide you by default. When using abstractions that allocate memory for you, keep in mind that they may reduce ASAN's effectiveness. The examples shown here were exclusively C++ examples, but can be easily applied to other languages as well.

Rust, for example, has zero ASAN englightenment at the time of this blog post. That means `unsafe { }` code manually reading from a `Vec<T>`'s data pointer or passing the pointer across an FFI boundary may run into similar false-negatives. Ditto for the `String` type, `OSString`, etc.