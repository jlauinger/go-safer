# go-safer

Go Vet-style linter to find incorrect uses of `reflect.SliceHeader` and `reflect.StringHeader`


## Incorrect usage patterns that are reported

`go-safer` reports the following two usage patterns:

 1. There is a composite literal of underlying type `reflect.SliceHeader` or `reflect.StringHeader`
 2. There is an assignment to an instance of type `reflect.SliceHeader` or `reflect.StringHeader` that was not created
    by casting an actual slice or `string`
    
Pattern 1 identifies code that looks like this:

```go
func unsafeFunction(s string) []byte {
    sH := (*reflect.StringHeader)(unsafe.Pointer(&s))
    bH := &reflect.SliceHeader{
        Data: sH.Data,
        Len:  sH.Len,
        Cap:  sH.Len,
    }
    return *(*[]byte)(unsafe.Pointer(bH)) 
}
```

It will also catch cases where `reflect.SliceHeader` has been renamed, like in `type MysteryType reflect.SliceHeader`.

Pattern 2 identifies code such as the following:

```go
func unsafeFunction(s string) []byte {
    strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
    sH := (*reflect.SliceHeader)(unsafe.Pointer(nil))
    sH.Len = strH.Len
    sH.Cap = strH.Len
    sH.Data = strH.Data
    return
}
```

`safer-go` will catch the assignments to an object of type `reflect.SliceHeader`. Using the control flow graph of the
function, it can see that `sH` was not derived by casting a real slice (here it's `nil` instead).

There are more examples on incorrect (reported) and safe code in the test cases in the `passes/sliceheader/testdata/src`
directory.


## Why are these patterns insecure?

If `reflect.SliceHeader` or `reflect.StringHeader` is not created by casting a real slice or `string`, then the Go runtime
will not treat the `Data` field within these types as a reference to the underlying data array. Therefore, if the garbage
collector runs just before the final cast from the literal header instance to a real slice or `string`, it may collect
the original slice or `string`. This can lead to an information leak vulnerability.

For more details, such as a Proof-of-Concept exploit and a suggestion for a fixed version of these unsafe patterns, read 
this blog post: [Golang Slice Header GC-Based Data Confusion on Real-World Code](https://dev.to/jlauinger/sliceheader-literals-in-go-create-a-gc-race-and-flawed-escape-analysis-exploitation-with-unsafe-pointer-on-real-world-code-4mh7)


## Install

To install `go-safer`, use the following command:

```
go get github.com/jlauinger/go-safer
```

This will install `go-safer` to `$GOPATH/bin`, so make sure that it is included in your `$PATH` environment variable.


## Usage

Run go-safer on a package like this:

```
$ go-safer example/cmd
```

Or supply multiple packages, separated by spaces:

```
$ go-safer example/cmd example/util strings
```

To check a package and, recursively, all its imports, use `./...`:

```
$ go-safer example/cmd/...
```

Finally, to check the package in the current directory you can use `.`:

```
$ go-safer .
```

`go-safer` accepts the same flags as `go vet`:

```
Flags:
  -V	print version and exit
  -all
    	no effect (deprecated)
  -c int
    	display offending line with this many lines of context (default -1)
  -cpuprofile string
    	write CPU profile to this file
  -debug string
    	debug flags, any subset of "fpstv"
  -fix
    	apply all suggested fixes
  -flags
    	print analyzer flags in JSON
  -json
    	emit JSON output
  -memprofile string
    	write memory profile to this file
  -source
    	no effect (deprecated)
  -tags string
    	no effect (deprecated)
  -trace string
    	write trace log to this file
  -v	no effect (deprecated)
```

Supplying the `-help` flag prints the usage information for `go-safer`:

```
$ go-safer -help
```


## Development

To get the source code and compile the binary, run this:

```
$ git clone https://github.com/jlauinger/go-safer
$ cd go-safer
$ go build
```

To run the test cases for the `sliceheader` pass, use the following commands:

```
$ cd passes/sliceheader
$ go test
```

`go-safer` uses the testing infrastructure from `golang.org/x/tools/go/analysis/analysistest`. To add a test case, create
a new package within the `bad` or `good` directories in `passes/sliceheader/testdata/src`. Add as many Go files to the
package as needed.

Then, register the new package in the `sliceheader_test.go` file by specifying the package path.

In the test case source files, add annotation comments to the lines that should be reported (or not). The comments must
look like this:

```
sH.Len = strH.Len            // want "assigning to reflect header object" "assigning to reflect header object"
fmt.Println("hello world")   // ok
```

Annotations that indicate a line that should be reported must begin with `want` and then have the desired message twice.
For some reason, the testing infrastructure will cause `go-safer` to output the annotation twice, therefore it has to be
expected twice as well to pass the test.


## License

Licensed under the MIT License (the "License"). You may not use this project except in compliance with the License. You 
may obtain a copy of the License [here](https://opensource.org/licenses/MIT).

Copyright 2020 Johannes Lauinger

This tool has been developed as part of my Master's thesis at the 
[Software Technology Group](https://www.stg.tu-darmstadt.de/stg/homepage.en.jsp) at TU Darmstadt.
