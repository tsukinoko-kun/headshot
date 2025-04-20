# Headshot

Similar to makeheaders but for C++. It generates a header file from a source file.

## Usage

Build hpp files once for a specified set of cpp files:

```
headshot <...cpp files>
```

or watch all directories and generate hpp files on the fly:  
*some directories are excluded (e.g., build, bin, vendored, .cache, ...)*

```
headshot watch
```
