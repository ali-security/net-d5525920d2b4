// Command zipper writes a module zip for the checked-out tree using Go's own
// module-zip writer, so the member set matches what proxy.golang.org serves.
//
// It lives under .github/ so that it is excluded from the module zip itself
// (the go tool also ignores directories whose name begins with a dot, so it
// never joins ./... builds) and is copied out to a scratch module at CI time.
package main

import (
    "log"
    "os"

    "golang.org/x/mod/module"
    "golang.org/x/mod/zip"
)

func main() {
    if len(os.Args) != 5 {
        log.Fatal("usage: zipper <module-path> <version> <source-dir> <output-zip>")
    }
    m := module.Version{Path: os.Args[1], Version: os.Args[2]}
    f, err := os.Create(os.Args[4])
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    if err := zip.CreateFromDir(f, m, os.Args[3]); err != nil {
        log.Fatal(err)
    }
    log.Printf("created module zip: %s", os.Args[4])
}
