# Headshot

Similar to makeheaders but for C++. It generates a header file from a source file.

## Usage

Build hpp files once for a specified set of cpp files:

```
headshot <...cpp files>
```

### Watch all directories and generate hpp files on the fly
*some directories are excluded (e.g., build, bin, vendored, .cache, ...)*

```
headshot watch
```

### Build all hpp files in all directories once

*some directories are excluded (e.g., build, bin, vendored, .cache, ...)*

```
headshot build
```

### Update the `headshot` binary

```
headshot update
```

### Ignore file

Protect files from a headshot by giving them a helmet ;)

Create a `.helmet` file in the root of your project (where you run `headshot build` or `headshot watch`) in the `.gitignore` syntax.

### Interface

Some declarations have to be written in the header file but not the implementation.
Use the `HEADSHOT_INTERFACE` define for that.

`main.cpp`:

```c++
#include "main.hpp"

#include <iostream>
#include <string>

#ifdef HEADSHOT_INTERFACE
struct {
        int num;
        std::string text;
} myStructure;
#endif  // HEADSHOT_INTERFACE

int main() {
        myStructure s;
        s.num = 42;
        s.text = "Hello, World!";
        std::cout << s.text << std::endl;
        return 0;
}
```

Generated `main.hpp`:

```c++
#ifndef MAIN_HPP
#define MAIN_HPP
#define HEADSHOT_INTERFACE_MAIN_HPP

#include <iostream>
#include <string>

#ifdef HEADSHOT_INTERFACE_MAIN_HPP
struct {
        int num;
        std::string text;
} myStructure;
#endif  // HEADSHOT_INTERFACE_MAIN_HPP

int main();

#undef HEADSHOT_INTERFACE_MAIN_HPP
#endif  // MAIN_HPP
```

### Formatting

Formatting is important because `headshot` is based on regex, not on a C++ parser.

An empty line between all functions, classes, and methods is required.

Use this `.clang-format`:

```yaml
---
BasedOnStyle: Google
IndentWidth: 8
InsertBraces: true
LineEnding: LF
KeepEmptyLinesAtEOF: false
KeepEmptyLinesAtTheStartOfBlocks: false
```
