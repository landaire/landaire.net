+++
title = "Deobfuscating World of Warships' Python Scripts"
description = "An in-depth analysis of how World of Warships obfuscates its game scripts and how to mostly deobfuscate them automatically."
summary = "An in-depth analysis of how World of Warships obfuscates its game scripts and how to mostly deobfuscate them automatically."
template = "toc_page.html"
toc = true
date = "2023-11-08"

[extra]
image = "/img/wows-obfuscation/header.png"
image_width =  250
image_height = 250
pretext = """
An in-depth analysis of how World of Warships obfuscates its game scripts and how to mostly deobfuscate them.
"""
+++

An in-depth analysis of how World of Warships obfuscates its game scripts and how to mostly deobfuscate them. The `wowsdeob` project can be found on GitHub: [https://github.com/landaire/wowsdeob](https://github.com/landaire/wowsdeob)

## Background

This blog post is something I'm writing 3 years after my initial research/development, and about 2 years after I stopped actively working on the tool. Some of the details in this blog may not be fully accurate from time slippage and a lot of the initial research notes I made were lost or scattered in Discord conversations.

I am only now writing this as the game is somewhat dead and the development team chooses to continually release content that makes gameplay worse. Submarines, hybrid battleships, aircraft carriers, and HE-spamming battleships with cruiser concealment that overmatch everything have led to a less enjoyable gameplay experience.

The tool and techniques have been kept to a very tight circle so that data mining could continue without a potential cat-and-mouse game with the developer. I apologize to that community if that does occur.

Two years ago I open-sourced a tool I called [`unfuck`](https://github.com/landaire/unfuck) which I described as a "Python 2.7 bytecode ~~deobfuscator~~ unfucker". That tool was the result of the work I had done for World of Warships, but isn't capable of completely deobfuscating World of Warships files out-of-the-box. `unfuck` [made the rounds on HN](https://news.ycombinator.com/item?id=28163546) where I got supportive responses such as:

> Look, I get that edgy names are fun, but I'm happy that I will never have to use this tool for work, and I pity the fool who has to explain why "unfuck" was needed to solve a real problem.

And:

>Ooof, really bad name. Makes me think the project or maintainer are immature...

And someone respecting my licensing choices:

>Interesting dual licensing

>> This project is dual-licensed under MIT and the ABSE ("Anyone But Stefan Esser") license. Note that an additional exception to the license is added, forbidding use/redistribution of said content to his trainees as well, but only when in a 5 mile radius from "Stefan Esser" or while holding any sort of (video)conference/chat with him.
>
>> Note that this license will only be used as long as what would capstone decode / that one other arm64 ida plugin thing by i0n1c ("Stefan Esser") are not under the MIT license. afterwards, all exceptions are cleared and basically MIT license applies

## What is World of Warships?

World of Warships (WoWs) is a free-to-play naval warfare multiplayer game released in 2015 by Wargaming and their Lesta Studio. It supports [multiple forms of modifications](https://web.archive.org/web/20230728140319/https://forum.worldofwarships.com/topic/174161-modapi-documentation/) via Adobe Flash for UI mods, XML/audio files for audio mods, custom ship textures, and Python for basically everything else. The Python APIs are somewhat limited, but can observe some in-game events that allow mod developers to surface information that the game doesn't show you by default.

Unlike a lot of multiplayer games, WoWs uses an authoritative server model where each client is only receiving events that the server believes they _should_ receive. For example, enemy ship locations and related information is only pushed down to clients when they are intended to be painted to your screen. There are no wall hacks and things like auto-aim are somewhat negated by player skill. That doesn't mean there aren't [illegal mods/cheats](https://v.youku.com/v_show/id_XNTgxNzIzNjAwNA==.html) that show you where to aim, where incoming artillery shells will land, etc.

With all of that said, there is very little incentive to cheat in this game. A top-tier player in World of Warships will have the game sense and skill that comes within a close margin to someone playing with cheats. Therefore deobfuscation of scripts doesn't necessarily help with cheat development apart from illegal mods, but may help modders create better mods in general.

## Game Scripts

Although the World of Warships developers are mostly transparent about how game mechanics work, there have still been some mysteries. For example, the [algorithm for how the dispersion ellipse of your shells is calculated](https://www.reddit.com/r/WorldOfWarships/comments/l1dpzt/reverse_engineered_dispersion_ellipse_including/) and the [dispersion of the shells themselves](https://www.reddit.com/r/WorldOfWarships/comments/mesoun/reverse_engineered_how_sigma_works_with_dispersion/).

The format of match replay files is also [mostly documented](https://github.com/Monstrofil/replays_unpack) through reverse engineering work of a few researchers and from reviewing publicly available engine source code, but there are still unknown elements to the serialized format. For these reasons it's highly desirable to review the implementation code to unmask these mysteries.

There has been at least one other individual who managed to deobfuscate and decompile the game scripts as far back as 2016. A World of Warships EU forum member by the name of ["TehRick" / "ThiSpawn" made multiple posts](https://web.archive.org/web/20230802004250/https://forum.worldofwarships.eu/topic/55613-understanding-wgs-armor-penetration-curves/?page=3) showing they had reverse engineered some of the C++ and Python logic. In fact, they explicitly called out the "Lesta anti-noob protection" obfuscated Python module name in one of their forum posts:

![TehRick forum post](/img/wows-obfuscation/tehrick_1.png)

In [a Reddit post](https://www.reddit.com/user/ThiSpawn) by the same username they also linked to a decompiled Python source file: [https://pastebin.com/y3Yk43Nd](https://pastebin.com/y3Yk43Nd).

Although TehRick seems to have disappeared soon after these posts, their code was still being referenced 4 years later!

*If you are TehRick / ThiSpawn feel free to reach out to me -- I would love to talk about your deobfuscator!*

## Python VM Primer

When Python source code is executed, you may notice that a `.pyc` file is created. The Python interpreter doesn't interpret raw source code -- it first compiles that source code to an intermediate representation that can be fed to the VM as instructions.

The VM is stack-based with some "registers" which are used for runtime storage of variables in pre-defined variable (or unnamed) slots. Almost every VM operation with the exception of loads/stores will directly modify values on the stack in some way. Variable registers cannot be modified in-place, and must first be put directly on the stack.

Each object on the stack or in a variable slot is a deserialized Python object representing one of the following types:

- None
- StopIteration
- Ellipsis
- Bool
- Long
- Float
- Complex
- Bytes
- String
- Tuple
- List
- Dict
- Set
- FrozenSet
- Code

Basically the raw primitive types you're probably used to when writing Python. All module definitions, functions, etc. are defined as `Code` objects that live in the `co_consts` section of their parent code object. Ditto with any lists, tuples, etc. that are hard coded in source code.

### Instructions

World of Warships uses Python 2.7 which has helpful documentation covering all instructions here: [https://docs.python.org/2/library/dis.html](https://docs.python.org/2/library/dis.html).

All instructions are at least 1 byte for the opcode and up to 3 bytes for instructions which have a 16-bit argument. For example, a `JUMP_ABSOLUTE 300` may be encoded as `0x71_012c` and a `POP_TOP` may be encoded as `0x01`.

The WoWs developers did not do any opcode remapping, which is a fairly common obfuscation trick when an application has the flexibility of embedding the Python VM.

## Prior Work on Deobfuscation

### lpcvoid's Findings

World of Warships' core game logic is contained within a `scripts.zip` file that is handled by a special loader. The loader reads compiled Python (files ending in `.pyc`) out of this zip archive and even uses a special technique of handling the serialized Python code object.

This logic and deobfuscating the _first stage_ of the matryoshka doll has already been described by [lpcvoid on his blog](https://lpcvoid.com/blog/0007_wows_python_reversing/index.html) and he even has a [part 2](https://lpcvoid.com/blog/0008_python_bytecode_dejunking/index.html) going into some of the junk instructions. I highly recommend reading his blog posts before continuing, as I will not rehash his hard work.

To summarize his findings: the module loader deserializes the bytecode object and uses the bytecode as an encryption key for some ciphertext stored in const data. After decrypting the ciphertext, the plaintext is decompressed (zlib) and is executed as a code object.

### Rapid Analysis of Bytecode Using pyasm

I wanted to take a quick moment to call out that my friend @gabe_k wrote a tool for a CTF challenge he put together many years ago called `pyasm` that is specifically designed for this type of scenario. It can gracefully handle disassembling of bad instructions, and even supports recompiling a `.pyasm` file back to a serialized code object. I've forked the project and made some quality of life improvements/bug fixes here: [https://github.com/landaire/pyasm](https://github.com/landaire/pyasm).

Here is an example of what one of these `.pyc` files look like when converted to a `.pyasm` file (redacted since it's very long):

```python
code
	stack_size 3
	flags 66
	consts 4
		none
		string "Wargaming.net | Lesta Studio"
		string "an error occurred while loading module"
		string "\x9a\xb6\x85A.^lPGO0Z/\xeeY3o\x11$,FCPCi&:U\x04\t\x02IS\x155EJ\x1a3\x96\x1f=*\xe2?\xa95\xa5\tf\x13hl\x92,\x12\x14T\x06\xc8o:\x08\x16\xd4\xfd\xd0\x8c\xc1T\xa0\x9b\xe5b\xc3%\x0eD\x8e\x85\xfb%\x83\x9b\xffL\rH\ruk\x14\xf2=\xd7\x86\t\x13\x0bk\x83HL\tH\xbf\x076IM\xa1\x0bTP\xdc\xc7<\n\x08\x8c\xdd\x05\'\x19\xf0\xa2FL\t\xbb\xd7\x15=\xd8\xc3\xba3\x19\x0fk\xd9\xe4\xf0\xbf\x9c?\nTJP\xc1\xcc\x17\xce\x04\x18\xfe98\x0e\x1d\x86\xdc\xab\x19\xe6)M,\x0e\xd4\xd8\xd4\x02\xb1\x0b(&;(o_\x1c\x1b\x16\xd8\xe1!-\xadw\xda%\xd0/\xa2\x1a\x08*7\xc7\x9d\x107\x0f\xe4\xcc\x1c\x0c)...
	end
	names 1
		string "locals"
	end
	instructions
		<255> 65532
		UNARY_INVERT 
		SETUP_EXCEPT 15
		LOAD_CONST 1 # Wargaming.net | Lesta Studio
		LOAD_NAME 0 # locals
		CALL_FUNCTION 0
		DUP_TOP 
		EXEC_STMT
		POP_BLOCK 
		JUMP_FORWARD 12
		3 * POP_TOP
		LOAD_CONST 2 # an error occurred while loading module
		PRINT_ITEM 
		PRINT_NEWLINE 
		JUMP_FORWARD 1
		END_FINALLY 
		LOAD_CONST 0 # None
		RETURN_VALUE 
		DELETE_NAME 23519
		UNARY_NOT
		DELETE_NAME 23438
		<216> 64603
		INPLACE_OR
		UNARY_NOT
		DELETE_NAME 23308
...
end
```

Throughout my analysis of the obfuscation tricks I frequently leaned on pyasm for manually reordering/deleting instructions in order to figure out which sections were causing hiccups for the decompiler.

## WoWs - Generic Obfuscation Tricks

### 1: Bogus Instructions

After decrypting and decompressing the 2nd stage code object, you'll find that tools like [uncompyle6](https://pypi.org/project/uncompyle6/) fail to decompile the bytecode to source with an error. In his 2nd blog post lpcvoid covered one of Lesta's tricks of inserting bogus instructions that contain completely invalid opcodes which confuse these types of tools. One thing not mentioned is that these instructions can also be valid but really mess up the VM's stack and cause static analysis to enter a bad state.

Unfortunately, tools like uncompyle are mostly intended to run on Python bytecode generated cleanly by a Python VM. Invalid opcodes are definitely not supported, and neither are invalid instruction operands. This trick is the most straightforward for causing a decompiler to choke.

### 2: Stack Reordering

Different Python code patterns -- even if semantically the same -- have very slight nuance in the emitted instructions. Tools like uncompyle in some cases rely on fragile instruction patterns for mapping to source code and will easily encounter errors when a wonky pattern is encountered.

The following is a completely made-up example, but consider the following instruction sequence:

```
              224  LOAD_CONST            0
              227  MAKE_FUNCTION_0       0  None

 L. 351       230  STORE_FAST            6  'f333'
```

This loads some code object from the const section, makes a function, and stores that function in `co_varnames[6]`. This would typically result in something like the following Python code:

```python
def f333():
    pass
```

Now imagine that the instructions were rewritten to push and immediately pop a value to the stack randomly in the middle of creating the function:

```
              224  LOAD_CONST            0
              227  MAKE_FUNCTION_0       0  None
              ...  LOAD_CONST            None
              ...  POP_TOP

 L. 351       230  STORE_FAST            6  'f333'
```

The `LOAD_CONST`/`MAKE_FUNCTION`/`STORE_FAST` pattern is broken by instructions that are effectively a no-op, and the signatures used by the decompilers are now broken.

### 3: Const Predicates

Related to trick #2, instruction patterns may be broken by inserting conditions that evaluate to a constant value. One side of the branch may bring you to some garbage instructions (trick #1) and on the other side of the branch will be the next set of valid code to be executed.

Consider the following pseudocode:

```python
              224  LOAD_CONST            0
              227  MAKE_FUNCTION_0       0  None
              if len({1, 2, 3} & {2}) > 0:
 L. 351       230  STORE_FAST            6  'f333'
              else:
              ...  POP_TOP
              ...  POP_TOP
              ...  POP_TOP
              ...  RETURN_VALUE

```

In the middle of defining a function we've inserted a check to see if two sets overlap, and if so  we store the function in variable slot #6 (`STORE_FAST 6`). If the sets do not overlap, we go down the bogus code path that screws up stack state.

### 4: Variable Renaming

This one really doesn't impact tools but definitely impacts any end user who manages to partially decompile anything: local variables and function names (in the serialized function object) are renamed to things that are illegal in Python source code such as keywords, operators, and a combination of these things with spaces. Function objects are typically renamed to be very large unique numbers. Here is a real example from when I first set on this project:

```python

            def f333(impf, f222):
                continue r ; = 66
                h >> ] = 87
                assert } break = 538
                l else try 6 = 199
                continue r ; += 66
                h >> ] -= 538
                assert } break *= 199
                if not continue r ; + h >> ] >= assert } break - h >> ] + l else try 6:
                    * k try 4 = 113
                * k try 4 = None
                2 c = 22
                v as lambda n = 433
                6 p = 147
                * k try 4 += 113
                2 c -= 433
                v as lambda n *= 147
                if not * k try 4 + 2 c >= v as lambda n - 2 c + 6 p:
                    pass
                (j in, % /= p, 6 { +, y [, += import r) = ()
```

### 5: Implicit Returns

The `RETURN_VALUE` instruction in Python returns the value located at the top of the VM's stack. Python will, as far as I'm aware, always emit a `RETURN_VALUE` with the immediately preceding instruction setting up the value to be returned. For example:

```
LOAD_FAST 0
RETURN_VALUE
```

This loads the value stored at `co_varnames[0]` to the top of stack and returns it. Splitting up these instructions will break the pattern decompilers use to transform this into `return varname`. Imagine if the in the following code `tos` literally represented the top of the stack (and not a variable slot):

```python
if condition:
    tos = get_return_value()
else:
    tos = other_return_value()

return
```

This would create the following control flow:

![Implicit return control flow](/img/wows-obfuscation/implicit_return.svg)

In this scenario the instruction immediately preceding the `RETURN_VALUE` isn't the instruction setting up the value -- it's likely some type of `JUMP` instruction or quite possibly anything else!

It's not clear to me if this is more of a code optimization trick, or an obfuscation trick, but really what's the difference?

### 6: Weird Jumps

Something else I noticed was that some jumps were just... weird? There were randomish-looking `JUMP_FORWARD N` instructions that didn't make sense (but usually were just jumping over garbage instructions), and were hard to disambiguate from those legitimately generated by the Python compiler. For example, the following Python code may insert a `JUMP_FORWARD 0` (jump to the next instruction):

```python
if !foo: # POP_JUMP_IF_FALSE
    # EMIT THIS CODE FIRST
    if foo:
        print "target"
    else:
        print "else"
# AFTER THE ABOVE BLOCK IS EMITTED, INSERT JUMP_FORWARD 0
else:
    # EMIT BLOCK
    print "main target"
```

This probably doesn't make much sense, so let me show a real example of the control flow at an instruction level:

{{ resize_image(path="/img/wows-obfuscation/unnecessary_jump.png", width=500, height=500, op="fit") }}


Do you notice the `JUMP_FORWARD 0` in the left-center node? It's completely unnecessary! The layout of these instructions when serialized is: `POP_TOP`, `POP_TOP`, `POP_TOP`, `JUMP_FORWARD 0`, `LOAD_FAST 2`. The execution sequence of these instructions is exactly the same as well. You could remove the `JUMP_FORWARD 0` and nothing of value would be lost. So why is it there?

It's just a side effect of how the Python compiler does codegen:

```python
def visitIf(self, node):
    end = self.newBlock()
    numtests = len(node.tests)
    for i in range(numtests):
        test, suite = node.tests[i]
        if is_constant_false(test):
            # XXX will need to check generator stuff here
            continue
        self.set_lineno(test)
        self.visit(test)
        nextTest = self.newBlock()
        self.emit('POP_JUMP_IF_FALSE', nextTest)
        self.nextBlock()
        self.visit(suite)
        self.emit('JUMP_FORWARD', end) # <--- UNCONDITIONALLY ADD `JUMP_FORWARD`
        self.startBlock(nextTest)
    if node.else_:
        self.visit(node.else_)
    self.nextBlock(end)
```

Sometimes that `JUMP_FORWARD` isn't jumping 0 bytes and may be some value that jumps over garbage instructions (and was inserted by the obfuscator). Other times this `JUMP_FORWARD` comes immediately after some other unconditional control flow instruction and will never be executed by a Python VM (or picked up by my instruction decoder).

So why can't it be removed if it's usually unnecessary? Unfortunately uncompyle relies on the presence of the `JUMP_FORWARD` to determine what type of condition has occurred.

### 7: Generally Weird Control Flow

This one is hard to express without sounding psychotic, but let me just say: **fuck loops, fuck exception handlers**. If you combine the weird jumps and false predicates in loops, you may be able to generate some code that looks like a loop but is only ever executed for one iteration before just jumping to some other part of the code because of a const predicate.

And what about exception handlers that intentionally raise an exception that's supposed to be caught to trigger the "good" code path?

And what about nesting exceptions inside of loops with const predicates?

After encountering such a scenario I was starting to feel like this:

![Meme of Charlie's conspiracy theory board from It's Always Sunny in Philadelphia](/img/wows-obfuscation/charlie.jpg)


## Deobfuscation

Deobfuscating everything together requires a solid framework that can achieve:

1. Parsing all instructions in a manner that doesn't break on bogus opcodes
2. Evaluating conditions to determine used/unused code
3. Restoring instruction ordering
4. Restoring variable names
5. Deoptimizing code (implicit returns)
6. Normalize weird control flow

...so yeah the natural path I took here was to rebuild the Python VM in Rust in 2 months and write somewhat spaghetti code that revisiting two years later has me thinking "wtf was I smoking?".

### Avoiding Bad Instructions

Parsing instructions is fairly straightforward: an instruction with no arguments is 1 byte (opcode), and an instruction with an argument is 3 bytes (opcode + uint16). You start parsing instructions by reading from offset 0 in the `co_code` section of the code object and just continue this in a loop.

Decompilers and disassemblers tend to read instructions **linearly** -- i.e. if the first instruction is `JUMP_ABSOLUTE 200` it's going to disassemble `JUMP_ABSOLUTE 200` from offset 0, then disassemble the next instruction from offset 3. This isn't great because you will run into a bunch of bogus instructions that can be avoided by simply creating a decoder that understands control flow.

To mitigate this in my deobfuscator I instead add instruction offsets to a queue. A `JUMP_ABSOLUTE 200` will add offset 200 as next in the queue, and a `JUMP_IF_{TRUE,FALSE} <TARGET>` will add the offset for the target **and** the next instruction to the queue.

Along the way I also compile an instruction graph where each node represents a basic block with edges to other basic blocks:

{{ resize_svg(path="/img/wows-obfuscation/stage2_obfuscated.svg", width=500, height=500) }}

### Removing const predicates

Remember how I said I rebuilt the Python VM in Rust? [I was serious, and it was _just_ to solve this problem.](https://github.com/landaire/unfuck/blob/de4c631aa725ff8da5aed8e718117b607c009c3d/src/smallvm.rs#L130) I do what I've called "partial execution". Certain builtin functions are handled, and an individual code object's opcodes are executed to obtain a snapshot of the VM's stack and perform taint tracking. The main function signature is:

```rust
/// Executes an instruction, altering the input state and returning an error
/// when the instruction cannot be correctly emulated. For example, some complex
/// instructions are not currently supported at this time.
pub fn execute_instruction<O: Opcode<Mnemonic = py27::Mnemonic>, F, T>(
    instr: &Instruction<O>,
    code: Arc<Code>,
    stack: &mut VmStack<T>,
    vars: &mut VmVars<T>,
    names: &mut VmNames<T>,
    globals: &mut VmNames<T>,
    names_loaded: LoadedNames,
    mut function_callback: F,
    access_tracking: T,
) -> Result<(), Error<O>>;
```

Brief rundown of all inputs:

- `instr`: the instruction to be executed
- `code`: the deserialized Python code object for which the instruction belongs to
- `stack`: current snapshot of the VM stack
- `vars`: all variable slots and their current values
- `names`: map to a `vars` slot
- `globals`: similar to `vars` but on a global level
- `names_loaded`: modules imported
- `function_callback`: callback for `CALL_FUNCTION` instruction and can be used for handling builtins or calling other code objects if desired
- `access_tracking`: data to associate with a VM stack value

When an instruction is executed and the stack is modified, the modified stack value will represent a tuple of `(Option<value>, [access_tracking])`. The first value in the tuple is the value that resulted from executing an instruction if it could be determined, otherwise `None`. The second value will represent some metadata for looking up instructions which contributed to generating/modifying that stack value (a basic block index + instruction index).

The execution loop for a basic block looks something like this:

```python
for instruction in basic_block:
    if instruction.is_conditional_jump():
        if tos[0].is_some():
            # Determine the truthiness of top-of-stack.
            #
            # Based off of the opcode (JUMP_IF_{TRUE,FALSE}), determine which
            # branch is never taken and remove the edge from this BB to the BB
            # we will never branch to.
            #
            # We also iterate tos[1] and remove all of the instructions which
            # contributed to tos[0].
    else:
        execute_instruction(instruction, ...)
```

This will effectively **remove** instructions related to const conditions and (usually) the basic block not taken from the graph. These inserted, fake basic blocks are implicitly removed since they become orphaned from the main code graph.

There are some issues with this approach:

1. I need to correctly implement almost all VM instructions
2. Large traces will balloon in computation complexity and memory
3. Complex arithmetic or unsupported instructions will immediately stop the VM, which leaves code only partially deobfuscated

### Restoring Variable Names

Variable names are unfortunately always lost. Instead of "restoring", I simply iterate and replace odd var names with the following script:

```python
def fix_varnames(varnames):
    global unknowns
    newvars = []
    for var in varnames:
        var = var.strip()
        unallowed_chars = '=!@#$%^&*()"\'/, '
        banned_char = False
        banned_words = ['assert', 'in', 'continue', 'break', 'for', 'def', 'as', 'elif', 'else', 'for', 'from', 'global', 'if', 'import', 'is', 'lambda', 'not', 'or', 'pass', 'print', 'return', 'while', 'with']
        for c in unallowed_chars:
            if c in var:
                banned_char = True

        if not banned_char:
            if var in banned_words:
                banned_char = True

        if banned_char:
            newvars.append('unknown_{0}'.format(unknowns))
            unknowns += 1
        else:
            newvars.append(var)
    
    return tuple(newvars)
```

Pretty simple replacement of illegal-looking variable name with `unknown_N`.

### Restoring Function Names

Unlike var names, function names generally _can_ be restored due to an oversight in the obfuscator. When a function is defined in a module, the bytecode for setting it up looks like the following:

```
LOAD_CONST <CONST_INDEX> # Load the code object
MAKE_FUNCTION # Take the loaded code object and tell the interpreter to turn it into a function
STORE_NAME <NAME_INDEX> # Store the created function at the specified named index
```

`NAME_INDEX` corresponds to a `name` string value located in the `co_names` array in the code object. The code object for the function also contains its name, which is generally what's used by decompilers to label a function. I leveraged this in my instruction handler loop by checking:

```python
for instruction in basic_block:
    if instruction.is_store_name():
        accessed_instructions = tos[1]
        if accessed_instructions[-1].is_make_function():
            # change the function name on the code object to match what this
            # scope sees as the function name
            fix_function_name(tos[0], co.co_names[instruction.argument])

    execute_instruction(instruction, ...)
```

Strictly speaking these names aren't even necessary for the VM to run the code since they're really only present for debugging purposes. I suspect that this is either an oversight by Lesta or they deliberately left these in for debugging crashes on clients (although I'm not certain if they send back Python crash reports). Maybe there's something I don't know though.

### Deoptimizing Code

I _think_ that `RETURN_VALUE` is really the only case I had to correct, and it was a fairly simple fix.

1. Look for basic blocks that contain only a `RETURN_VALUE` instruction
2. For each incoming edge, replace the final instruction in the basic block to be `RETURN_VALUE`
3. Remove the basic block from step 1

The example control flow for return values then provided [in the section about implicit returns](#5-implicit-returns) will now look like this:

![Implicit return fixups](/img/wows-obfuscation/implicit_return_fix.svg)

### Fixing Bad Instructions

There are some scenarios where code paths containing bad instructions can't be outright removed. Usually it's because a condition couldn't be proven to be a const predicate, or there was some other factor involved that led to it not being removed. Even though I _know_ they're bad, I'm hesitant to outright outright remove the nodes and conditions as gaps in the VM and data mixing may lead to incorrect instruction removal.

To correct these basic blocks I calculate the depth of the stack at the location of the bad instruction, insert enough `POP_TOP` instructions to clear out the stack, and finally put a `LOAD_CONST None` and `RETURN_VALUE` at the end of the basic block to force a `return None`.

## The Rest of The Matryoshka Doll

There are four distinct "stages" to loading the Python module, two of which we've already loosely discussed from lpcvoid's blog (encrypted code, and the compressed code).

### Stage 2 - Decompressed Code

The following is an example stage 3 payload:

```python
import sys, marshal, copy_reg
if id(marshal.loads) != copy_reg.mmId:
    return
code = sys._getframe().f_back.f_code.co_code
impf = (isinstance(__builtins__, dict) or __builtins__).__import__ if 1 else __builtins__['__import__']
if not hasattr(impf, 'func_code') or hash(impf.func_code.co_code) != 1236377808:
    if hasattr(impf, 'func_code') and type(impf.func_globals['common']) != type(marshal.loads):
        return

    def f123--- This code section failed: ---
                0  LOAD_GLOBAL           0  'common'
                3  LOAD_FAST             0  'arg'
                6  LOAD_FAST             1  'kw'
                9  CALL_FUNCTION_VAR_KW_0     0  None
               12  STORE_FAST            2  'res'
               15  LOAD_GLOBAL           1  'type'
               18  LOAD_FAST             2  'res'
               21  CALL_FUNCTION_1       1  None
               24  LOAD_ATTR             2  '__name__'
               27  LOAD_CONST               'module'
               30  COMPARE_OP            2  ==
               33  POP_JUMP_IF_FALSE   155  'to 155'
               36  LOAD_GLOBAL           3  'hasattr'
               39  LOAD_FAST             2  'res'
               42  LOAD_CONST               '__file__'
               45  CALL_FUNCTION_2       2  None
               48  POP_JUMP_IF_FALSE   155  'to 155'
               51  LOAD_GLOBAL           3  'hasattr'
               54  LOAD_FAST             2  'res'
               57  LOAD_CONST               'gCPLBx86'
               60  CALL_FUNCTION_2       2  None
               63  UNARY_NOT
               64  POP_JUMP_IF_TRUE     82  'to 82'
               67  LOAD_FAST             2  'res'
               70  LOAD_ATTR             4  'gCPLBx86'
               73  LOAD_CONST               '1663084375'
               76  COMPARE_OP            3  !=
             79_0  COME_FROM            64  '64'
               79  POP_JUMP_IF_FALSE   155  'to 155'
               82  LOAD_FAST             0  'arg'
               85  LOAD_CONST               0
               88  BINARY_SUBSCR
               89  LOAD_CONST               ('collections', 'utf8_test', 'copy_reg')
               92  COMPARE_OP            7  not-in
             95_0  COME_FROM            79  '79'
             95_1  COME_FROM            48  '48'
             95_2  COME_FROM            33  '33'
               95  POP_JUMP_IF_FALSE   155  'to 155'
               98  SETUP_EXCEPT         48  'to 149'
              101  LOAD_GLOBAL           5  'loaded'
              104  LOAD_ATTR             6  'add'
              107  LOAD_GLOBAL           7  'id'
              110  LOAD_FAST             2  'res'
              113  CALL_FUNCTION_1       1  None
              116  LOAD_FAST             0  'arg'
              119  LOAD_CONST               0
              122  BINARY_SUBSCR
              123  BUILD_TUPLE_2         2
              126  CALL_FUNCTION_1       1  None
              129  POP_TOP
              130  LOAD_GLOBAL           8  'sys'
              133  DUP_TOP
              134  LOAD_ATTR             9  'errCnt'
              137  LOAD_CONST               1
              140  INPLACE_ADD
              141  ROT_TWO
              142  STORE_ATTR            9  'errCnt'
              145  POP_BLOCK
              146  JUMP_FORWARD          6  'to 155'
            149_0  COME_FROM            98  '98'
              149  POP_TOP
              150  POP_TOP
              151  POP_TOP
              152  JUMP_FORWARD          0  'to 155'
            155_0  COME_FROM           152  '152'
            155_1  COME_FROM           146  '146'
              155  LOAD_FAST             2  'res'
              158  RETURN_VALUE
               -1  RETURN_LAST

Parse error at or near `None' instruction at offset -1


    def f123(impf, f222):
        import sys
        f222.func_globals['loaded'] = set()
        if not isinstance(__builtins__, dict):
            f222.func_globals['common'] = __builtins__.__import__ if 1 else __builtins__['__import__']
            f222.func_globals['sys'] = sys
            sys.errCnt = 0
            __builtins__.__import__ = isinstance(__builtins__, dict) or f222
        else:
            __builtins__['__import__'] = f222
        sys.settrace(None)
        sys.settrace = sys.getrefcount
        sys.setprofile(None)
        sys.setprofile = sys.getrefcount
        sys.gettrace = sys.exit
        sys.getprofile = sys.exit
        return


    f333(impf, f222)
    impf = f222

def f123(marshaled):
    swapMap = {0: 151, 1: 235, 2: 9, 3: 249, 4: 100, 5: 10, 6: 188, 7: 106, 8: 128, 9: 122, 10: 220, 11: 189, 12: 242, 13: 253, 14: 210, 15: 243, 16: 5, 17: 27, 18: 222, 19: 90, 20: 139, 21: 18, 22: 79, 23: 255, 24: 230, 25: 83, 26: 20, 27: 74, 28: 89, 29: 141, 30: 219, 31: 123, 32: 203, 33: 51, 34: 98, 35: 53, 36: 103, 37: 204, 38: 190, 39: 118, 40: 62, 41: 161, 42: 41, 43: 241, 44: 247, 45: 101, 46: 196, 47: 153, 48: 181, 49: 40, 50: 152, 51: 174, 52: 140, 53: 171, 54: 44, 55: 134, 56: 158, 57: 88, 58: 70, 59: 132, 60: 173, 61: 2, 62: 129, 63: 8, 64: 86, 65: 21, 66: 148, 67: 145, 68: 211, 69: 127, 70: 224, 71: 167, 72: 185, 73: 237, 74: 147, 75: 233, 76: 58, 77: 175, 78: 14, 79: 252, 80: 209, 81: 155, 82: 37, 83: 162, 84: 42, 85: 227, 86: 78, 87: 136, 88: 12, 89: 246, 90: 81, 91: 126, 92: 186, 93: 6, 94: 87, 95: 150, 96: 96, 97: 39, 98: 193, 99: 28, 100: 55, 101: 59, 102: 200, 103: 30, 104: 225, 105: 197, 106: 212, 107: 213, 108: 245, 109: 179, 110: 105, 111: 111, 112: 112, 113: 32, 114: 156, 115: 91, 116: 68, 117: 50, 118: 13, 119: 66, 120: 84, 121: 159, 122: 182, 123: 102, 124: 221, 125: 154, 126: 57, 127: 254, 128: 130, 129: 17, 130: 82, 131: 77, 132: 104, 133: 95, 134: 146, 135: 48, 136: 169, 137: 164, 138: 121, 139: 223, 140: 11, 141: 232, 142: 244, 143: 218, 144: 85, 145: 113, 146: 177, 147: 166, 148: 52, 149: 24, 150: 170, 151: 4, 152: 73, 153: 144, 154: 236, 155: 34, 156: 205, 157: 115, 158: 114, 159: 226, 160: 45, 161: 234, 162: 19, 163: 133, 164: 168, 165: 135, 166: 194, 167: 99, 168: 138, 169: 251, 170: 46, 171: 72, 172: 60, 173: 94, 174: 31, 175: 75, 176: 3, 177: 178, 178: 116, 179: 238, 180: 7, 181: 143, 182: 92, 183: 142, 184: 176, 185: 25, 186: 108, 187: 250, 188: 16, 189: 160, 190: 107, 191: 240, 192: 208, 193: 0, 194: 187, 195: 49, 196: 15, 197: 184, 198: 199, 199: 43, 200: 165, 201: 38, 202: 125, 203: 76, 204: 110, 205: 71, 206: 33, 207: 217, 208: 1, 209: 229, 210: 120, 211: 131, 212: 195, 213: 69, 214: 231, 215: 97, 216: 248, 217: 201, 218: 206, 219: 22, 220: 23, 221: 35, 222: 207, 223: 124, 224: 137, 225: 65, 226: 157, 227: 93, 228: 180, 229: 56, 230: 117, 231: 63, 232: 191, 233: 109, 234: 239, 235: 36, 236: 202, 237: 163, 238: 119, 239: 214, 240: 183, 241: 54, 242: 172, 243: 29, 244: 47, 245: 228, 246: 198, 247: 61, 248: 26, 249: 149, 250: 67, 251: 216, 252: 192, 253: 80, 254: 64, 255: 215}
    marshaled = ('').join(map(chr, [ swapMap[ord(n)] for n in marshaled ]))
    return marshaled


co_code = [ chr(((byte ^ 38) & 126 | (byte ^ 38) >> 7 & 1 | ((byte ^ 38) & 1) << 7) ^ 89) for byte in [ ord(byte) for byte in f123(code) ] ]
locDict = {}
locDict['globs'] = sys._getframe().f_back.f_globals
locDict['code'] = marshal.loads(('').join(co_code[::-1]))
locDict['marshal'] = marshal
exec locDict['code'] in locDict

def f111():
    pass


f111()
del f111
```
There are some checks to ensure that certain state is set up, but in general this will:

1. Load the `co_code` from the original stage 1 file
2. Apply a substitution cipher over each byte
3. Do some bit arithmetic on each byte of the result from step 2
4. Execute the code that was just decoded

In my deobfuscator I was able to leverage the custom Python VM to apply the swapmap for me by creating a state machine. Essentially I scan for certain instructions that look like they're applying the swapmap, then execute that function with some fake VM stack set up. That code can be found here: [https://github.com/landaire/wowsdeob/blob/ffeeedaea9390c1d1e9ba785360e75aaa1aa10d0/src/smallvm.rs](https://github.com/landaire/wowsdeob/blob/ffeeedaea9390c1d1e9ba785360e75aaa1aa10d0/src/smallvm.rs)

### Stage 3

This stage is pretty boring all things considered. The Stage 3 code object is just another compressed code object that's been base64 encoded and had the result reversed.

![Sample showing the base64-encoded data](/img/wows-obfuscation/stage3_base64.png)

No big tricks here. The deobfuscator logic can be found here: [https://github.com/landaire/wowsdeob/blob/ffeeedaea9390c1d1e9ba785360e75aaa1aa10d0/src/main.rs#L290-L297](https://github.com/landaire/wowsdeob/blob/ffeeedaea9390c1d1e9ba785360e75aaa1aa10d0/src/main.rs#L290-L297).

Worth noting that this is the stage which references the Lestas "Anti noobs protection"!

![Anti noobs protection](/img/wows-obfuscation/stage3_anti_noobs_protection.png)

### Stage 4

We have the final module! This module now needs to have all generic deobfuscation tricks applied to get decompilation.

## End Result

The end result of this effort is going from a file that fails to decompile:

```bash
❯ uncompyle6 ./output/AirplaneUtils_stage4.pyc
# uncompyle6 version 3.8.0
# Python bytecode 2.7 (62211)
# Decompiled from: Python 2.7.18 (default, Sep 28 2022, 20:52:16)
# [GCC Apple LLVM 14.0.0 (clang-1400.0.29.102)]
# Warning: this version of Python has problems handling the Python 3 byte type in constants properly.

# Embedded file name: 26977129990194521
# Compiled at: 2020-12-14 08:10:48
Traceback (most recent call last):
  File "/Users/lander/.pyenv/versions/2.7.18/bin/uncompyle6", line 10, in <module>
    sys.exit(main_bin())
  File "/Users/lander/.pyenv/versions/2.7.18/lib/python2.7/site-packages/uncompyle6/bin/uncompile.py", line 194, in main_bin
    **options)
  File "/Users/lander/.pyenv/versions/2.7.18/lib/python2.7/site-packages/uncompyle6/main.py", line 328, in main
    do_fragments,
  File "/Users/lander/.pyenv/versions/2.7.18/lib/python2.7/site-packages/uncompyle6/main.py", line 230, in decompile_file
    do_fragments=do_fragments,
  File "/Users/lander/.pyenv/versions/2.7.18/lib/python2.7/site-packages/uncompyle6/main.py", line 149, in decompile
    co, out, bytecode_version, debug_opts=debug_opts, is_pypy=is_pypy
  File "/Users/lander/.pyenv/versions/2.7.18/lib/python2.7/site-packages/uncompyle6/semantics/pysource.py", line 2578, in code_deparse
    co, code_objects=code_objects, show_asm=debug_opts["asm"]
  File "/Users/lander/.pyenv/versions/2.7.18/lib/python2.7/site-packages/uncompyle6/scanners/scanner2.py", line 350, in ingest
    pattr = names[oparg]
IndexError: tuple index out of range
zsh: exit 1     uncompyle6 ./output/AirplaneUtils_stage4.pyc
```

To:

```python
# uncompyle6 version 3.8.0
# Python bytecode 2.7 (62211)
# Decompiled from: Python 2.7.18 (default, Sep 28 2022, 20:52:16) 
# [GCC Apple LLVM 14.0.0 (clang-1400.0.29.102)]
# Warning: this version of Python has problems handling the Python 3 byte type in constants properly.

# Embedded file name: 123823449462
# Compiled at: 2020-12-14 08:10:48
import math, random
from ConstantsUtils import idGenerator
import GameParams, Junk, Lesta
from Math import Vector3
from AirPlanes.AirplaneConstants import DeathReason, SquadronStateEnum, Throttle, TurnDirection, PlaneTypeNames
from AirplaneConstants import SQUADRON_DEPARTURE_BIT, SQUADRON_PURPOSE_BIT, SQUADRON_INDEX_BIT, PLANETYPE_2_PARAMSNAME, PLANE_TORPEDO_CONE_HALF_WIDTH, PlaneTypes, PLANE_PROJECTILE_GRAVITY
from mc0f1198d import devMode
from md0ce06f9 import LOG_ERROR
from m79622f13 import normaliseAngle, getDirectionFromYaw, lerp, lerpAngles, getDirectionFromYawPitch, EPSILON
from PlanesDEFConverter import PlanesDictConverter
from PyMagic import pTuple
from mc062022a import ShipTypes
from shared_constants.m22c5a818 import PLANE_AMMO_TYPES

class WayPoint:
    enum = idGenerator(0)
    GENERATED = next(enum)
    RESET = next(enum)
    LAUNCHING_START_NODE = next(enum)
    LAUNCHING_END_NODE = next(enum)
    LANDING_START_NODE = next(enum)
    LANDING_END_NODE = next(enum)
    del enum

    def __init__(self, pos, yaw, pitch, time, waypointType=GENERATED):
        self.pos = pos
        self.yaw = yaw
        self.pitch = pitch
        self.time = time
        self.sent = False
        self.type = waypointType

    def __repr__(self):
        return ('<< Waypoint pos:{0}, time:{1}, type:{2}, yaw:{3}, pitch: {4}>>').format(self.pos, self.time, self.type, self.yaw, self.pitch)

    def toDict(self):
        return {'position': Vector3(self.pos), 'yaw': self.yaw, 'pitch': int(normaliseAngle(self.pitch, False) * 127 / math.pi), 'time': int(self.time * 1000), 'type': self.type}

    @staticmethod
    def fromDict(dict):
        return WayPoint(Vector3(dict['position']), dict['yaw'], dict['pitch'] * math.pi / 127.0, dict['time'] / 1000.0, dict['type'])

    @staticmethod
    def _splineReference(point1, point2, t):
        unknown_0 = point1.pos.flatDistTo(point2.pos)
        if unknown_0 > 0:
            unknown_1 = 1.0 - t
            unknown_2 = unknown_0 * getDirectionFromYaw(point1.yaw)
            1 = unknown_0 * getDirectionFromYaw(point2.yaw)
            unknown_3 = unknown_1 * unknown_1 * (1.0 + 2.0 * t) * point1.pos + t * t * (1.0 + 2.0 * unknown_1) * point2.pos + unknown_1 * unknown_1 * t * unknown_2 - unknown_1 * t * t * 1
            unknown_4 = point1.pos.flatDistTo(unknown_3) / unknown_0
            unknown_3.y = lerp(point1.pos.y, point2.pos.y, unknown_4)
            unknown_5 = lerpAngles(point1.yaw, point2.yaw, t)
            unknown_6 = 0.0
        else:
            unknown_3 = lerp(point1.pos, point2.pos, t)
            unknown_5 = lerpAngles(point1.yaw, point2.yaw, t)
            unknown_6 = 0.0
        return (unknown_3, unknown_5, unknown_6)

    def spline(point1, point2, t):
        return Lesta.splineWayPoints(point1.pos, point1.yaw, point1.pitch, point2.pos, point2.yaw, point2.pitch, t)

    spline = _splineOptimised


def generateSquadronId(shipId, index, purpose, departureId):
    """
        Generates id of the squadron based on the given arguments.
        :param shipId: id of the owner
        :type shipId: int
        :param index: index of the squadron within the owner
        :type index: int
        :param purpose: the function that squadron is performing
        :type purpose: int (AirplaneConstants.SquadronPurpose)
        :param departureId: unique departure id of the squadron incremented with every subsequent id generation
        :type departureId: int
        :return id of the squadron
        :rtype: int
        """
    unknown_7 = departureId << SQUADRON_DEPARTURE_BIT | purpose << SQUADRON_PURPOSE_BIT | index + 1 << SQUADRON_INDEX_BIT | shipId
    assert retrieveOwnerID(unknown_7) == shipId
    assert retrieveSquadronIndex(unknown_7) == index
    assert retrieveSquadronPurpose(unknown_7) == purpose
    return unknown_7


def retrieveSquadronIndex(squadronId):
    unknown_8 = 7
    return (squadronId >> SQUADRON_INDEX_BIT & unknown_8) - 1


def retrieveSquadronPurpose(squadronId):
    unknown_9 = 7
    return squadronId >> SQUADRON_PURPOSE_BIT & unknown_9


def retrieveSquadronDeparture(squadronId):
    unknown_10 = 7
    return squadronId >> SQUADRON_DEPARTURE_BIT & unknown_10


def parseSquadronId(squadronId):
    return (retrieveSquadronIndex(squadronId), retrieveSquadronPurpose(squadronId), retrieveSquadronDeparture(squadronId))


def retrieveOwnerID(id):
    return id & 4294967295L


def getPlaneName(params, planeType):
    unknown_11 = PLANETYPE_2_PARAMSNAME.get(planeType)
    if planeType:
        return params.__dict__[unknown_11].planeType
    return planeType


def getTorpedoingArea--- This code section failed: ---
                0  LOAD_GLOBAL           0  'getDirectionFromYaw'
                3  LOAD_FAST             1  'attackDir'
                6  LOAD_ATTR             1  'yaw'
                9  LOAD_GLOBAL           2  'math'
               12  LOAD_ATTR             3  'pi'
               15  LOAD_CONST               2
               18  BINARY_DIVIDE    
               19  BINARY_ADD
               20  CALL_FUNCTION_1       1  None
               23  STORE_FAST            6  'unknown_14'
               26  LOAD_GLOBAL           4  'min'
               29  LOAD_CODE                <code_object 369464740902>
               32  MAKE_FUNCTION_0       0  None
               35  LOAD_FAST             4  'formation'
               38  LOAD_ATTR             5  'positions'
               41  GET_ITER         
               42  CALL_FUNCTION_1       1  None
               45  CALL_FUNCTION_1       1  None
               48  STORE_FAST            7  'unknown_15'
               51  LOAD_GLOBAL           6  'max'
               54  LOAD_CODE                <code_object 369537115186>
               57  MAKE_FUNCTION_0       0  None
               60  LOAD_FAST             4  'formation'
               63  LOAD_ATTR             5  'positions'
               66  GET_ITER         
               67  CALL_FUNCTION_1       1  None
               70  CALL_FUNCTION_1       1  None
               73  STORE_FAST            8  'unknown_16'
               76  LOAD_FAST             8  'unknown_16'
               79  LOAD_FAST             7  'unknown_15'
               82  BINARY_SUBTRACT  
               83  STORE_FAST            9  'unknown_17'
               86  LOAD_FAST             9  'unknown_17'
               89  LOAD_FAST             5  'currentPlaneCount'
               92  BINARY_MULTIPLY  
               93  LOAD_FAST             4  'formation'
               96  LOAD_ATTR             7  'npositions'
               99  BINARY_DIVIDE    
              100  STORE_FAST           10  'unknown_18'
              103  LOAD_FAST             0  'attackPoint'
              106  LOAD_FAST            10  'unknown_18'
              109  LOAD_CONST               2
              112  BINARY_DIVIDE    
              113  LOAD_GLOBAL           8  'PLANE_TORPEDO_CONE_HALF_WIDTH'
              116  BINARY_SUBTRACT  
              117  LOAD_FAST             6  'unknown_14'
              120  BINARY_MULTIPLY  
              121  BINARY_SUBTRACT  
              122  STORE_FAST           11  'unknown_19'
              125  LOAD_FAST             0  'attackPoint'
              128  LOAD_FAST            10  'unknown_18'
              131  LOAD_CONST               2
              134  BINARY_DIVIDE    
              135  LOAD_GLOBAL           8  'PLANE_TORPEDO_CONE_HALF_WIDTH'
              138  BINARY_ADD       
              139  LOAD_FAST             6  'unknown_14'
              142  BINARY_MULTIPLY  
              143  BINARY_ADD       
              144  STORE_FAST           12  'unknown_20'
              147  LOAD_FAST            11  'unknown_19'
              150  LOAD_FAST            12  'unknown_20'
              153  COMPARE_OP            2  ==
              156  POP_JUMP_IF_FALSE   176  'to 176'
              159  LOAD_FAST            12  'unknown_20'
              162  LOAD_FAST             6  'unknown_14'
              165  LOAD_CONST               0.001
              168  BINARY_MULTIPLY  
              169  INPLACE_ADD      
              170  STORE_FAST           12  'unknown_20'
              173  JUMP_FORWARD          0  'to 176'
            176_0  COME_FROM           173  '173'
              176  LOAD_FAST             0  'attackPoint'
              179  LOAD_FAST             1  'attackDir'
              182  LOAD_FAST             2  'planeParams'
              185  LOAD_ATTR             9  'torpedoAimDist'
              188  BINARY_MULTIPLY  
              189  BINARY_ADD       
              190  STORE_FAST           13  'unknown_21'
              193  LOAD_FAST            13  'unknown_21'
              196  LOAD_FAST             6  'unknown_14'
              199  LOAD_FAST             3  'spreading'
              202  BINARY_MULTIPLY  
              203  LOAD_CONST               0.5
              206  BINARY_MULTIPLY  
              207  BINARY_SUBTRACT  
              208  STORE_FAST           14  'unknown_22'
              211  LOAD_FAST            13  'unknown_21'
              214  LOAD_FAST             6  'unknown_14'
              217  LOAD_FAST             3  'spreading'
              220  BINARY_MULTIPLY  
              221  LOAD_CONST               0.5
              224  BINARY_MULTIPLY  
              225  BINARY_ADD       
              226  STORE_FAST           15  'unknown_23'
              229  LOAD_FAST            11  'unknown_19'
              232  LOAD_FAST            12  'unknown_20'
              235  LOAD_FAST            14  'unknown_22'
              238  LOAD_FAST            15  'unknown_23'
              241  BUILD_TUPLE_4         4 
              244  RETURN_VALUE     
               -1  RETURN_LAST      

Parse error at or near `None' instruction at offset -1


def getBombingZone(planeParams, modifierParams, aimAccuracy, attackerStrength=1.0):
    unknown_24 = planeParams.maxSpread
    unknown_25 = planeParams.minSpread
    unknown_26 = lerp(unknown_24[0], unknown_25[0], aimAccuracy) * modifierParams.planeSpreadMultiplier
    unknown_27 = lerp(unknown_24[1], unknown_25[1], aimAccuracy) * modifierParams.planeSpreadMultiplier
    0 = planeParams.outerSalvoSize[0] * unknown_26 * attackerStrength
    unknown_28 = planeParams.outerSalvoSize[1] * unknown_27
    unknown_29 = planeParams.innerSalvoSize[0] * unknown_26 * attackerStrength
    unknown_30 = planeParams.innerSalvoSize[1] * unknown_27
    return (0, unknown_28, unknown_29, unknown_30)


# snip
```

Clearly some functions still fail to decompile, but it may be enough to just read the instructions at this point to understand the intent.

And code objects that go from this:

{{ resize_svg(path="/img/wows-obfuscation/simple_obfuscation_example.svg", width=500, height=500) }}


To this:

{{ resize_svg(path="/img/wows-obfuscation/simple_deobfuscation_example.svg", width=500, height=500) }}

## Closing Thoughts

There's one common theme in this post that I hope some readers picked up on: we are constantly battling the _decompiler's_ ability to unravel code based off of heuristics instead of battling the obfuscator injecting garbage. The const predicates for example aren't even that big of a deal -- they just insert false control flow that at a _source_ level is fairly straightforward to see is garbage.

However, this false control flow is enough to throw off the decompiler's ability to figure out a _single_ source code pattern that results in the entire code object failing to decompile. This isn't a jab at `uncompyle` either -- it's a great tool that works fairly well considering there's zero competition in this space. However, I think that if I were to solve this problem from scratch in 2023 I'd solve it very differently by working on a better decompiler that erodes away mapping of 1:1 source code to bytecode, and instead focuses on rebuilding source with the same functional semantics.

As reverse engineers we don't really care about how the code was originally written, we just want to understand its intent at a higher level.

## Thanks

Thank you, reader, for making it this far. I'd like to extend thanks to the following people for their support in this research/providing deobfuscator feedback:

- lpcvoid (without his initial blog post I wouldn't have been nearly as motivated to go down this endeavour)
- Track
- TTaro
- 901234
- notyourfather
- gabe_k
- Scout1Treia
- EdibleBug