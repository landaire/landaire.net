+++
title = "Memory Safety by Default is Non-Negotiable"
description = "\"And suppose once more, that he is reluctantly dragged up a steep and rugged ascent, and held fast until he is forced into the presence of the sun himself, is he not likely to be pained and irritated? When he approaches the light his eyes will be dazzled, and he will not be able to see anything at all of what are now called realities.\""
summary = "\"And suppose once more, that he is reluctantly dragged up a steep and rugged ascent, and held fast until he is forced into the presence of the sun himself, is he not likely to be pained and irritated? When he approaches the light his eyes will be dazzled, and he will not be able to see anything at all of what are now called realities.\""
template = "toc_page.html"
toc = true
date = "2026-07-14"
+++

The year is 2026. The Programming Language Holy Wars are still going, and [the much anticipated blog post](https://bun.com/blog/bun-in-rust) from Bun (JavaScript runtime) creator Jarred Sumner about using Claude to translate the code from Zig to Rust has sent people into a frenzy.

Before continuing any further: if you have a hobby project which you choose to write in `$lang` because you enjoy it, and are aware it has some sharp edges that can cut you from time to time, this is probably not addressed to you. Continue enjoying writing code in that language and ignore people like me who say it's not good enough.

But if you are writing critical infrastructure (browsers, media codecs, networking stacks, etc.) and believe that you've dulled those edges: you are wrong. You will still get cut in ways you didn't predict, and the blade extends to cut your users too.

## The Consequences of Memory Corruption

Surveillance-for-hire is a big industry targeting many groups with the goal of intercepting communication and providing access to information which may otherwise be difficult to obtain. I do not have exact numbers to quantify, but a _lot_ of spyware I've seen begins its life on a device with a memory safety issue.

Of the many groups impacted by spyware, I'll focus on just one: **journalists**. There are many documented cases of journalists whose harm (death, imprisonment, etc.) has been confirmed, or strongly believed, to have been linked to spyware use against them:

- [Jamal Khashoggi](https://en.wikipedia.org/wiki/Assassination_of_Jamal_Khashoggi)
- [Cecilio Pineda Birto](https://www.theguardian.com/news/2021/jul/18/revealed-murdered-journalist-number-selected-mexico-nso-client-cecilio-pineda-birto)
- [Khadija Ismayilova](https://en.wikipedia.org/wiki/Khadija_Ismayilova)
- [Ahmed Mansoor](https://en.wikipedia.org/wiki/Ahmed_Mansoor)

And others who have "only" been targeted/infected, possibly putting people in their lives (including sources) at risk:

- [Ben Hubbard](https://citizenlab.ca/research/breaking-news-new-york-times-journalist-ben-hubbard-pegasus/)
- [Galina Timchenko](https://citizenlab.ca/pegasus-infection-of-galina-timchenko-exiled-russian-journalist-and-publisher/)
- [Szabolcs Panyi and András Szabó](https://vsquare.org/szabolcs-panyi-i-was-hacked-with-pegasus-software/)
- [22 members of El Faro staff](https://beta.elfaro.net/en/el-salvador-en/22-members-of-el-faro-bugged-with-spyware-pegasus)
- [100 journalists/members of civil society using WhatsApp](https://www.theguardian.com/technology/2025/jan/31/whatsapp-israel-spyware)

[...the list goes on](https://www.amnesty.org/en/latest/press-release/2021/07/the-pegasus-project/), and these are only the ones which have been both discovered _and_ reported on.

While the difficulty of exploiting memory safety issues has risen to tip the scales against being economical for cybercrime groups, it still happens. [In 2017 WannaCry and other ransomware](https://en.wikipedia.org/wiki/WannaCry_ransomware_attack) re-used the leaked [`EternalBlue`](https://en.wikipedia.org/wiki/EternalBlue) exploit to devastate networks, including that of the UK's National Health Service, causing disruptions in patient care and sending IT departments into chaos.

Recently some groups have also done [watering hole attacks](https://en.wikipedia.org/wiki/Watering_hole_attack) against iOS users by using ["leaked" spyware kits for the purposes of stealing cryptocurrency](https://iverify.io/blog/coruna-inside-the-nation-state-grade-ios-exploit-kit-we-ve-been-tracking).

## Lipstick on a Pig

I have personally worked on several investigations involving mercenary spyware. From the reporting in the previous section and my own experience, some form of memory corruption vulnerability is exploited to achieve remote code execution on someone's device nearly every time. These are sometimes chained with a logic bug which can be classified as a vulnerability, but on its own has limited impact.

[Time](https://projectzero.google/2026/05/pixel-10-exploit.html) and [time](https://projectzero.google/2026/01/pixel-0-click-part-1.html) and [time](https://projectzero.google/2025/12/android-itw-dng.html) and [time](https://projectzero.google/2025/05/breaking-sound-barrier-part-i-fuzzing.html) and [time](https://projectzero.google/2025/03/blasting-past-webp.html) and [time](https://projectzero.google/2023/10/an-analysis-of-an-in-the-wild-ios-safari-sandbox-escape.html) and [time](https://projectzero.google/2023/09/analyzing-modern-in-wild-android-exploit.html) and [time](https://projectzero.google/2022/03/forcedentry-sandbox-escape.html) again, it has been proven that even with state-of-the-art hardware and software exploit mitigations on locked down devices, attackers are still getting in.

The last 10 years of my professional career in infosec has covered everything from web applications, hypervisors, kernels, "secure" kernels, and mobile applications.

Could you guess which from that list is the only stack that's really meaningfully moved the needle against remote code execution in the last 10 years?

Web applications!

Why? Because RCEs in this domain are 99.9999% of the time _logic bugs_ commonly caused by using unsafe deserializers / command injection, so people simply moved away from or hardened those.

[Smashing the Stack for Fun and Profit](https://phrack.org/issues/49/14) was released almost 30 years ago -- just over a year after I was born. My career 30 years later is based on the same principles and ideas in that blog post, but shifting towards fancier patterns of the same bug.

With that said, AI is [still finding and exploiting linear stack buffer overflow bugs in widely-used operating systems](https://github.com/califio/publications/blob/main/MADBugs/CVE-2026-4747/write-up.md). Although to be fair, Address Space Layout Randomization [is relatively new technology from 2001](https://pax.grsecurity.net/docs/aslr.txt). FreeBSD implemented it in user-mode in 2019 so who knows, maybe in 2037 they'll add kASLR and `-fstack-protector-strong`.

Even with mitigations proven to be effective, [some projects refuse to adopt them](https://marc.info/?l=freebsd-arch&m=143239155805369&w=2). We have invested so many resources into trying to un-fuck C and C++, the primary languages used in the other domains (and mobile for performance-sensitive applications), that at this point it's starting to grow old.

<!--## Eliminating Vulnerability Classes

My journey through programming so far in life has looked something like this:

- ~2008: C#
- ~2010: C, C++
- ~2011: Python, Go, Java, Ruby
- ~2013: PHP, JavaScript, (entire web stack basically)
- ~2014: Swift
- ~2015: D, Rust

My main general-purpose languages over time were C#, then Go, and then around 2016 I finally started to really go all-in on learning Rust which has been my language of choice ever since. I didn't even really appreciate or care about the borrow checker. I liked the features of the language and got my ass kicked by the borrow checker constantly.

But as I used it more I started to appreciate that I wasn't chasing subtle bugs that had me questioning if I had a logic issue or memory corruption issues. If my code compiled, it could only be a logic bug ([barring soundness bugs in the compiler/libraries](https://github.com/rust-lang/rust/issues?q=state%3Aopen%20label%3AI-unsound)).

After using Rust for so long, I truly do not understand why anyone would not want their programming language to **eliminate** as many issues as possible at compilation time and ensure that the programmer cannot even introduce certain classes of bugs into their program.

Yes, there are real tradeoffs to this such as the mental energy fighting a borrow checker when you aren't used to it, and re-architecting code to appease it. Not to mention possible runtime costs to e.g. bounds checks that cannot be automatically elided by the compiler.-->

## Organizational Cost of Memory Safety Issues

As a security engineer who has focused professionally on _native_ security for the past 8 years, I've gotten to see first-hand the impact memory corruption issues have on engineers and organizations.

It's not terribly uncommon to see a completely random spike in native crashes that the on-call production engineer has to look at. The root cause might be:

1. As simple as "oh there's a stack-use-after-return in the function at the top of the stack frame introduced in the last release".
2. As non-obvious as "we introduced a new field into this packed struct which threw off its alignment and we were triggering a `SIGBUS` on a misaligned pointer only on 32-bit `$ARCH`".
3. As challenging as chasing down a racy UaF in code that's completely untouched but only now surfaced because the teardown of another thread got shorter and objects allocated to a single arena were mistakenly shared across threads.
4. Something random like the underlying OS API changing in ways we didn't properly account for.
5. My personal favorite that is never obvious at all to SWEs: _oops, someone introduced another cross-DSO (dynamic shared object) indirect function call and didn't add the caller to the sanitizer ignore list, so [CFI](https://clang.llvm.org/docs/ControlFlowIntegrity.html) `SIGTRAP`'d_.

Not surprising, but security teams also periodically find issues that warrant immediate attention from the product team. There are months though where nothing interesting is found and suddenly over the course of a couple weeks you're finding new issues every day.

Product teams are obviously randomized by this as they work to fix issues, but their efforts don't end as soon as a bug is fixed. Incident reviews, knowledge sharing, fixing variants, meetings up the chain of management to express urgency and the need to shift resources, investing in secure frameworks, etc. all have cognitive and time costs.

Sometimes these issues would randomize engineers for weeks to months, causing genuine thrash in a team and preventing valuable team resources from helping with more meaningful feature or architectural work.

[Andrew Kelley in his response to the Bun blog post said](https://andrewkelley.me/post/my-thoughts-bun-rust-rewrite.html#:~:text=There%27s%20a%20dichotomy,of%20those%20things.):

> There's a dichotomy being presented here where you have to either choose a "style guide" or a programming language feature in order to avoid bugs. The sleight of hand misdirects the reader away from the main way bugs are eliminated: by dedicating engineering resources to it. You're not giving TigerBeetle nearly enough credit. Quite simply they put in the time to find and eliminate the bugs, they make an effort to maintain a healthy relationship with ZSF, and Bun did neither of those things.

Having a good relationship with the language authors should not be a required step on the pathway to writing bug-free code. You also shouldn't have to hire people whose job it is to repair cracks created by subpar tools (which, plainly speaking, is what most native security engineers including myself do).

You are also not going to be able to write idiomatic code all the time. You can hire world-class people, do a very painful review cycle to force idiomatic patterns, a style guide, etc., yet bugs still get through (see: Chromium).

At the end of the day, the only people who want to go chasing memory corruption issues are security engineers. _Nobody_ -- including security engineers -- wants to spend their time and energy fixing memory corruption issues.

## Memory-Unsafe Apologists

I cannot necessarily understand why new programming languages are being created that do not, from the start, make as many classes of bugs impossible to write **by default** and require clear annotations for the escape hatches. C# and Rust forcing you to use the `unsafe` keyword makes the sketchy stuff extremely obvious, and in Rust lets you tell the borrow checker "trust me, I know what I'm doing here" when you really need to without completely throwing out safety guarantees -- even in that unsafe block.

If you are designing a programming language that only mitigates these issues if someone _opts in_, you are doing everyone a disservice unless you are **very** honest that the language is fundamentally unsafe. Don't make up bullshit about "uhh well there's a couple safety features _if you compile in this mode_" because in practice people aren't going to use them.

C and C++ have taught us that there are plenty of people out there (a majority, probably) who believe they are too good to pay the perf cost of hardening their runtime. libc++ hardening has existed for YEARS and very, very few people opt in. `-fsanitize=bounds` has existed for some time (at least 2019?). Does anyone use it? My company does in our pre-release channels! Or what about `-ftrivial-auto-var-init=zero`/`-D_FORTIFY_SOURCE=$N`?

Going back to Bun's rewrite from Zig to Rust, I've seen some interesting takes on the topic like:

- > [Dude all they had to do was write idiomatic Zig. There's nothing Rust is doing at the JSC FFI boundary that Zig can't.](https://xcancel.com/Aut4rk/status/2075436514977132574) ... [And the entire problem they ran into is they weren't careful with the JSC FFI boundary. Literally all they had to do was use defer correctly, and use arena allocators scoped to JSC contexts.](https://xcancel.com/Aut4rk/status/2075441183237501232)
- > [What you need is architecture your programs around a few independent arenas, but this is not what people learned and it requires more whiteboard thinking.](https://ziggit.dev/t/rewriting-bun-in-rust/16592/4)
- > [Not sure if many people noticed that we now have a SafeAllocator in 0.17\[...\] All leaks will be reported. All double free and operation races will panic or segfault. Most write after free will panic or segfault.](https://ziggit.dev/t/rewriting-bun-in-rust/16592/16)

### "All they had to do was write idiomatic Zig"

What is idiomatic Zig? I'm not a Zig developer, so I'll read some of the guide looking specifically for bits about how use-after-frees -- one of the classes plaguing Bun -- can be mitigated. There's conveniently [a section in their documentation covering lifetimes and ownership](https://ziglang.org/documentation/0.16.0/#Lifetime-and-Ownership).

The first line says:

> It is the Zig programmer's responsibility to ensure that a pointer is not accessed when the memory pointed to is no longer available. Note that a slice is a form of pointer, in that it references other memory.

Ah. So _you_ are solely responsible for your own safety. Well, C++ for example has smart pointers which, when used correctly, can help _a lot_ so maybe Zig has something similar? **No.** Not according to the docs -- which isn't surprising since it competes with C rather than C++.

Ok, so I'll try to trigger a UaF with a slice:

```zig
const std = @import("std");

pub fn main() !void {
    var debug_allocator: std.heap.DebugAllocator(.{}) = .init;
    defer _ = debug_allocator.deinit();
    const gpa = debug_allocator.allocator();

    // create a list with an exact capacity of 4 so that pushing the 5th element
    // triggers a re-allocation
    var list = try std.ArrayList(u32).initCapacity(gpa, 4);
    defer list.deinit(gpa);
    while (list.items.len < list.capacity) {
        list.appendAssumeCapacity(@intCast(list.items.len * 10 + 10));
    }

    // `view` shares the list's current backing buffer (same ptr, same memory)
    const view = list.items;
    std.debug.print("before: ptr={*} cap={d} view={any}\n", .{ view.ptr, list.capacity, view });

    // force a realloc of the original list
    try list.append(gpa, 99);
    std.debug.print("after:  ptr={*} cap={d}\n", .{ list.items.ptr, list.capacity });

    // 💥
    std.debug.print("uaf:    view[0]={d} view={any}\n", .{ view[0], view });
}
```

This prints the following with no compiler errors or warnings:

```
An error occurred:
before: ptr=u32@7f12fbb00000 cap=4 view={ 10, 20, 30, 40 }
after:  ptr=u32@7f12fbae0000 cap=38
Segmentation fault at address 0x7f12fbb00000
/tmp/playground2265792951/play.zig:25:64: 0x113d9f9 in main (play.zig)
    std.debug.print("uaf:    view[0]={d} view={any}\n", .{ view[0], view });
                                                               ^
/home/play/.zvm/0.15.2/lib/std/start.zig:627:37: 0x113e499 in posixCallMainAndExit (std.zig)
            const result = root.main() catch |err| {
                                    ^
/home/play/.zvm/0.15.2/lib/std/start.zig:232:5: 0x113d291 in _start (std.zig)
    asm volatile (switch (native_arch) {
    ^
???:?:?: 0x0 in ??? (???)
```

As far as I can tell, it's idiomatic Zig. Nothing prevents me from aliasing a pointer to something that can be re-allocated, or warns me about it, just like C or C++. Even though this occurs in a single scope and should be statically detectable...

### "Use arenas!"

I don't know why people suddenly got so excited about arenas, but I've seen them mentioned quite a bit. Arenas are useful when you have a group of objects with a common lifetime, possibly consisting of very small allocations or allocations which you know the complete size of ahead of time.

Arenas basically bundle the lifetime of a bunch of objects together with the expectation that you can throw them away together, but don't magically make complex state safe unless arena allocations NEVER cross over with memory allocations outside of that arena. Even then, who's guaranteeing?

Imagine the following scenario: you have some data that gets allocated in an arena, and it needs to escape to the main heap. So, forgetting that the object has pointer fields, you do a shallow copy to the general heap.

This example doesn't even trigger a UaF/segmentation fault using the debug allocator, nor is it observable:

```zig
const std = @import("std");

const Inner = struct {
    value: u64,
};

const Outer = struct {
    id: u32,
    inner: *Inner,
};

fn makeDanglingObject(
    backing_allocator: std.mem.Allocator,
    output_arena: std.mem.Allocator,
) !*Outer {
    var temporary_arena_state =
        std.heap.ArenaAllocator.init(backing_allocator);
    defer temporary_arena_state.deinit();

    const temporary_arena = temporary_arena_state.allocator();

    // Both objects initially live in the temporary arena.
    const inner = try temporary_arena.create(Inner);
    inner.* = .{
        .value = 0x1234_5678,
    };

    const outer = try temporary_arena.create(Outer);
    outer.* = .{
        .id = 42,
        .inner = inner,
    };

    std.debug.print(
        "inside function: outer={*} inner={*} value=0x{x}\n",
        .{ outer, outer.inner, outer.inner.value },
    );

    // The returned Outer itself lives in the caller-provided output arena.
    const result = try output_arena.create(Outer);

    // Shallow copy: result.inner still points into temporary_arena.
    result.* = outer.*;

    return result;

    // temporary_arena_state.deinit() runs here.
    // result remains alive in output_arena, but result.inner is dangling.
}

pub fn main() !void {
    var debug_allocator: std.heap.DebugAllocator(.{
// these change nothing

//        .safety = true,
//        .never_unmap = true,
//        .retain_metadata = true,
    }) = .init;
    defer _ = debug_allocator.deinit();

    const gpa = debug_allocator.allocator();
//const gpa = std.heap.page_allocator;

    var output_arena_state = std.heap.ArenaAllocator.init(gpa);
    defer output_arena_state.deinit();

    const output_arena = output_arena_state.allocator();

    const object = try makeDanglingObject(gpa, output_arena);

    std.debug.print(
        "after return:    outer={*} inner={*}\n",
        .{ object, object.inner },
    );

    var temporary_arena_state =
        std.heap.ArenaAllocator.init(gpa);
    defer temporary_arena_state.deinit();

    const temporary_arena = temporary_arena_state.allocator();

    // Both objects initially live in the temporary arena.
    const inner = try temporary_arena.create(Inner);
    inner.* = .{
        .value = 0x41414141,
    };
    std.debug.print(
        "new inner:       {*} value=0x{x}\n",
        .{ inner, inner.value },
    );

    // 💥
    std.debug.print(
        "uaf:             id={d} value=0x{x}\n",
        .{ object.id, object.inner.value },
    );
}
```

With debug allocator:

```
inside function: outer=test.Outer@1043a0020 inner=test.Inner@1043a0018 value=0x12345678
after return:    outer=test.Outer@1043a0098 inner=test.Inner@1043a0018
new inner:       test.Inner@1043a0118 value=0x41414141
uaf:             id=42 value=0x12345678
```

On my machine it still does not hard fault using `std.heap.page_allocator` for the `gpa`, but it is observable:

```
inside function: outer=test.Outer@100dac020 inner=test.Inner@100dac018 value=0x12345678
after return:    outer=test.Outer@100db0018 inner=test.Inner@100dac018
new inner:       test.Inner@100dac018 value=0x41414141
uaf:             id=42 value=0x41414141
```

So the answer people will say is to design the structure like this where the object owns its allocator:

```
const Outer = struct {
    id: u32,
    inner: *Inner,
    arena: *std.heap.ArenaAllocator
};
```

It's better, but this has still bit people in C codebases I've worked on and, again, is not a strictly enforceable pattern.

### "Use safe allocators!"

I'll interpret this two ways: using a _secure_ allocator and using a _checked_ allocator.

Full ASAN comes with compile-time instrumentation which makes it very powerful and detect small out-of-bounds reads/writes, use-after-frees, double-frees, and other classes of issues, but of course comes with the tradeoff of increasing `.text` size, runtime costs, and memory costs. Unacceptable for production, but tolerable for CI/tests and possibly pre-production environments.

[Zig's debug allocator](https://github.com/ziglang/zig/blob/738d2be9d6b6ef3ff3559130c05159ef53336224/lib/std/heap/debug_allocator.zig#L5-L23) only detects, at this time, double-frees and leaks. This debug allocator seems to not surface the clear use-after-free as a side effect of never re-using the same address twice.

Using [enhanced-security allocators](https://docs.webkit.org/Deep%20Dive/Libpas/Libpas.html#overview) can be a good middle-ground in production for isolating allocations in problematic code and forcing fastfail by using e.g. guard pages and strategies that immediately unmap pages after they've been freed. But they come with costs you may not want to pay in release builds:

1. Additional overhead (syscalls like `mmap()` when allocating new pages).
2. Additional memory overhead from allocation metadata.
3. Separate allocation _behaviors_ in a different subsystem of the application. Allocations may incur non-obvious side effects from a standard allocator.
4. You need to exercise issues in order for them to be detected and fixed. Even when triggered, do you really have enough data to root-cause the bug?
5. You could be making mitigations like [MTE](https://security.apple.com/blog/memory-integrity-enforcement/) more challenging to implement for your custom allocator, if not impossible.

### "FFI boundary is the problem"

Some of the specific issues called out in the [Rewriting Bun in Rust blog post](https://bun.com/blog/bun-in-rust) are difficult to find PRs for, so I tried my best to find what I _think_ were the issues.

- [node:zlib: refuse reset() while a write is in progress](https://github.com/oven-sh/bun/pull/33523)
- [node:zlib: don't abort when the native handle is driven outside the zlib.ts lifecycle](https://github.com/oven-sh/bun/pull/32762)
- [zlib: make brotli/zstd native handle close() idempotent](https://github.com/oven-sh/bun/pull/32382)
- [udp: root sendMany payloads in MarkedArgumentBuffer / reorder send() to prevent UAF](https://github.com/oven-sh/bun/pull/30057)
- [udp: fix heap OOB in sendMany when connect state changes mid-iteration](https://github.com/oven-sh/bun/pull/29851)
- [MessageEvent: name Locker so m_data lock is actually held](https://github.com/oven-sh/bun/pull/30290)

And some other ones:

- [fix(websocket): double-free on ws.close() during proxy TLS handshake](https://github.com/oven-sh/bun/pull/29101)
- [fix(node:fs): refactor StatWatcher to use JSRef instead of indirect Strong ref](https://github.com/oven-sh/bun/pull/28065)
- [Fix crash when a client aborts a streaming response under backpressure](https://github.com/oven-sh/bun/pull/32120)
- [Fix out-of-bounds read in JS highlighter on trailing backslash in `${}`](https://github.com/oven-sh/bun/pull/31435)
- [Fix out-of-bounds write in toUTF16Alloc sentinel branch](https://github.com/oven-sh/bun/pull/29982)
- [fix(url): out-of-bounds write in Bun.pathToFileURL with long relative paths](https://github.com/oven-sh/bun/pull/29688)
- [strings: fix out-of-bounds read in CodepointIterator.next() on truncated UTF-8](https://github.com/oven-sh/bun/pull/29999)

These can be roughly categorized as:

| Class                           | Example                                                                                                                                                       | Fixed by...                                                                                                               |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Type confusion                  | [#32120](https://github.com/oven-sh/bun/pull/32120)                                                                                                           | Adding new var to separate out userdata for callbacks                                                                     |
| Improper state management       | [#33523](https://github.com/oven-sh/bun/pull/33523), [#29851](https://github.com/oven-sh/bun/pull/29851)                                                      | Blocking improper state transitions, capturing state which might change                                                   |
| Dangling references             | [#30057](https://github.com/oven-sh/bun/pull/30057), [#29101](https://github.com/oven-sh/bun/pull/29101), [#28065](https://github.com/oven-sh/bun/pull/28065) | Hoisting an object to keep a reference alive and prevent GC/operation reordering, clearing state, changing reference type |
| Improper thread synchronization | [#30290](https://github.com/oven-sh/bun/pull/30290)                                                                                                           | Mutex was a temporary which was dropped immediately... in C++?                                                            |
| Out-of-bounds access            | [#31435](https://github.com/oven-sh/bun/pull/31435), [#29982](https://github.com/oven-sh/bun/pull/29982)                                                      | Adding a bounds check                                                                                                     |

I think people are sort of right and wrong here: as things _currently_ stand, I think Rust only solves improper thread synchronization, some dangling reference issues, and _maybe_ the type confusion. This assumes janky raw pointer patterns aren't being ported over to Rust though.

Just to state it plainly: from the _huge_ sample size of N=10-15 PRs I looked at which fixed memory corruption issues, Bun did not seem to focus on building solid abstractions outside some TypeScript scripts which did codegen, and I think Bun could have been made much more reliable in pure Zig with better design patterns. I'm not reading the full code though so it's very hard to say comprehensively.

It seems to me like the code was not designed with a clear FFI layer involved. Although not uncommon in other languages, I feel like this pattern is somewhat forced in Rust to avoid going crazy:

```
┌───────────────────────┬─────────────────────┬─────────────────────────────┬┐
│                       │                     │                             ││
│                       │                     │                             ││
│             ◀─────────┼───────▶             │                             ││
│                       │                     │                             ││
│                       │                     │                             ││
│                       │                     │                             ││
│   Idiomatic primary   │Primary language code│                             ││
│  language code which  │ which encapsulates  │      3P Language Code       ││
│knows nothing about FFI│         FFI         │                             ││
│                       │                     │                             ││
│                       │                     │                             ││
│                       │            ◀────────┼────────▶                    ││
│                       │                     │                             ││
│                       │                     │                             ││
│                       │                     │                             ││
└───────────────────────┴─────────────────────┴─────────────────────────────┴┘
```

But in the PRs above I looked at, some of the Zig code seems to skip the middle box and kind of intermix things.

That said, I still think Rust is the right choice long-term.

Zig folks say that being explicit is better than implicit -- but the tradeoff is that you _will_ forget to do things. The answer of "just remember to `defer`!" doesn't guarantee the bugs will never occur. I can explicitly `defer` to ensure that some child resource is de-initialized, but forget to clear out the pointer. That's not a deterministically preventable bug.

_But you can design things poorly in Rust too! And have functions in Zig which handle proper cleanup of resources for you!_

Zig (or C) can't _force_ it to be that way with comprehensive state cleanup. C++ kind of can if you ban all the APIs that escape raw pointers.

Rust generally has a lot of friction when you do things the _wrong_ way such that you are ~~forcefully~~ naturally guided to do things the right way. Right now as things stand Bun seems to still be doing things the wrong way IMO but considering their goal was 1:1 port, these will hopefully be fixed over time.

For example, you can design resource management in Rust to be pretty simple:

```rust
struct JscRef(*mut c_void);

impl Drop for JscRef {
    fn drop(&mut self) {
        jsc_release_ref(self.0);
    }
}

impl Clone for JscRef {
    fn clone(&self) -> JscRef {
        jsc_retain_ref(self.0);

        JscRef(self.0)
    }
}

struct SomeRefHolder {
    child_resource: Option<JscRef>,
}
```

Now when you drop a reference to `SomeRefHolder` or do `self.child_resource.take()`, it internally drops the reference. Cloning it will increase the ref count. Outside of the module in which `JscRef` is defined, you cannot alias the held object pointer.

Zig is pretty similar to C and "smart objects" like this are not possible to express in Zig as far as I know. You must explicitly call functions which mimic this behavior, but nothing is protecting against accidental and inappropriate aliasing of the inner pointer.

Here's a good litmus test for whether the language is doing the bare minimum: can you design a mutex which makes it _impossible_ to do unsynchronized access to the data it's guarding without resorting to unconventional tricks like reflection, `unsafe`, and without shoving the data in an opaque pointer labeled "don't touch me unless you know what you're doing"? If the answer is "no", there are other related patterns which cannot be expressed for proper safety (e.g. smart pointers).

### "AI can't find the bugs?"

Yeah man the classic "Use non-deterministic tooling to drive the bugs to zero, rewrite with Best Practices(TM) and you'll never have bugs again if you write careful code". You're so right. Works every time.

## "But what about..."

### ASAN

Runtime cost. Code bloat from instrumentation.

[GWP-ASAN](https://llvm.org/docs/GwpAsan.html) requires surprisingly careful thought on execution to ensure it's not sampling allocations which have long-lived lifetimes, and may not work well on resource-constrained devices.

Both ASAN and GWP-ASAN require exercising the bug to detect it and are not going to deterministically _prevent_ or _mitigate_ bugs. It's a large-scale ITW bug finding tool by treating your users as natural fuzzers.

### Fil-C

Cool project. Not a huge fan of the creator repeatedly saying "safer than Rust" even if it's true in some ways. They are in different categories to me. It's like saying "AoT C# is safer than Rust!". Seems like he says it to ragebait people or something. Last I checked it wasn't objectively overall safer, and didn't prevent [intra-object overflows](https://github.com/google/sanitizers/wiki/AddressSanitizerIntraObjectOverflow).

Seems like a decent choice if you for some reason don't care about perf, need memory safety, and cannot vibeslop the codebase to a better language.

### Sandboxing

Your pig has found a new, clean sty to roam in.

I think sandboxing is a perfectly acceptable solution when you don't want to pay the cost of the alternatives: using a library in a safe language which is less feature-complete, owning a rewrite in a memory-safe language, or it's a black box (binary blob) you genuinely cannot do anything about. It has performance costs and, most critically, generally doesn't prevent someone from getting code execution.

It might make code execution more difficult to achieve **and** more difficult to access the resources a bad actor wants -- but neither is impossible in this scenario. Cross-compiling to WASM or some other _interpreted_ format I'd classify as an effective sandbox though which, in all likelihood, can make both impossible but may not be viable for performance reasons.

### Compiler/Hardware Mitigations

More lipstick that has measurable cost (`perf|binary size|hardware complexity`) is what we need. Surely. I sometimes wonder what an entirely memory-safe software stack with none of: stack cookies, CFI, PAC, MTE, hardened allocators, ASLR, etc. would look like in terms of perf.

What's the comprehensive cost we've grown to accept over time? How many billions of dollars are spent on power consumed executing stack cookie checks?

I can appreciate however that people _are_ putting resources into compiler and hardware mitigations for difficult problems for code which cannot change.

### Aerospace/Automotive Industry Uses C and C++ 😡

At this point I'm just ranting but [go watch Laurie's video](https://www.youtube.com/watch?v=Gv4sDL9Ljww). Understand too that they also use custom compilers which are certified for their needs.

## So... what do you like?

Here are my thoughts:

| Lang          | Rating | Explanation                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| C             | 👎     | I have audited completely new, clean codebases from world-class C programmers that still had a decent number of exploitable bugs. Thankfully the codebases are usually relatively straightforward to review. I don't think C gives enough _enabled-by-default_ analysis tooling to help you with complex applications and the language design does not allow for certain patterns that benefit the programmer.                                                                                                                                                                                                                    |
| Zig           | 👎     | Is the explicit nature of the language really a positive long-term for sufficiently complex codebases? I don't think so. It's C with some quality-of-life features, and new errors for... unused local variables? Lightweight lifetime analysis would be huge. Why would I pick this over another natively-compiled language with features that enable secure patterns? I think it's a bit telling that in his talk, ["Software Should Be Perfect"](https://www.youtube.com/watch?v=Z4oYSByyRak), Andrew Kelley never mentions "use-after-free" and most of the talk over-indexes on allocation failures as the root of all evil. |
| Odin          | 👎     | Bounds checking is pretty deeply drilled into people's brains at this point and better patterns like `for-in` loops have emerged. While appreciated, I think it could be argued lifetime safety is more prevalent these days in complex applications and it's unfortunate to see nothing to help with this.                                                                                                                                                                                                                                                                                                                       |
| Objective-C   | 👎     | [nemo wrote all about exploiting Objective-C](https://phrack.org/issues/69/9). Give it a read. A lot of this has been hardened over time, but Obj-C still allows a lot of unsafe things. For example, I've found many stack-use-after-return bugs in Objective-C code.                                                                                                                                                                                                                                                                                                                                                            |
| C++           | 🤷‍♂️     | I'd rather build on a C++ codebase that makes minor use of templating than a C codebase. At least I can abuse the complexities of C++ to make writing bugs more difficult. I'd rather _review_ a C codebase though.                                                                                                                                                                                                                                                                                                                                                                                                               |
| Go            | 👍     | Kind of whatever. I used to enjoy the language around 2013 but my opinions shifted negatively over time. Very narrow, uncommon opportunities for mem corruption though. GC'd anyways and not going to be used in most domains where Rust / Zig / C / C++ apply.                                                                                                                                                                                                                                                                                                                                                                   |
| Swift         | 👍     | Complex in a different way -- too many keywords and bolted-on features. Apple has invested in trying to make Swift the one-size-fits-all solution _after the fact_. It's not for me, personally, but provides good memory safety guarantees and modern features.                                                                                                                                                                                                                                                                                                                                                                  |
| Rust          | 👍👍   | Do I need to explain?                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Jai           | ❓     | For all intents and purposes, this language only exists for Jonathan Blow and to solve problems unique to him. It's not even publicly available.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Mojo / Carbon | ❓     | No real thoughts as I've not seen any adoption of these languages, and am not familiar with the concrete safety guarantees they intend to provide.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |

## Closing Thoughts

The Bun rewrite blog post isn't the first time this topic crossed my mind. But the community cope of "If only they did things this very `(niche|less performant|design that fundamentally changes the entire application architecture)` way and it solves the problem, how could they be so stupid?!" drives me a bit insane.

You should be asking _why it would not solve their problems_ rather than asserting it does.

People are also ignoring that sometimes you just straight up have to write some slop (even by hand) to move fast and get things done. I think it's very rare that people can write objectively great code that is probably the best solution in a first pass.

You might be screaming, _"That's why I didn't choose Rust in the first place! Velocity! Fighting the borrow checker!"_ and I hear you. But I truly believe that your alternative should be able to protect you in all the same ways Rust does, with an escape hatch for the very small and narrowly-scoped bits that need to be extremely hand-tuned.

Languages that don't [are losing a battle](https://www.theregister.com/software/2025/03/02/c-creator-calls-for-action-to-address-serious-attacks/1401456) for a place in production. Engineers and organizations are tired of participating in the endless cycle of fixing memory safety issues.

Rust may not be _the_ answer. But memory safety should not be an afterthought. Rust is simply the best choice on the safety-to-performance axis at this time.

You can make arguments that it's difficult to port a huge, complex codebase like ffmpeg, JavaScriptCore, Chromium, Firefox, ntoskrnl.exe, etc. and therefore we need to give them tools still to make the languages saf**er** where possible and reduce the impact of bugs (ASAN, MTE, PAC). I think that's fair.

Not to mention, there are target platform requirements that make it difficult to write anything but C or C++. I can't compile Rust for the Xbox 360 (yet), so even I'm forced to write C or C++ if I want to write some hacks.

But no _new_ critical codebases should be written in these languages if it can be avoided. The cost across many axes -- including genuine harm to users who have _nothing_ to do with the code side of things -- is unacceptable.

Things like Ghostty and TigerBeetle (Zig projects) aren't really critical stacks in the sense that they are on an attack surface **and** critical. I mean this when I say it: have fun building in your favorite programming language for these types of applications.

Bun I'd argue _is_ critical since it has builtin libraries for processing network requests/images/compression/crypto and, as a runtime, who knows how people are going to use the language itself.

The Chromium Project's [Rule of 2](https://chromium.googlesource.com/chromium/src/+/HEAD/docs/security/rule-of-2.md) puts it best:

{{ figure(src="/img/unsafe-languages/rule_of_two.png", caption="The Rule of 2") }}
