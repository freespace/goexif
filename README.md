DecodeFile()
============

This fork of `goexif` introduces `tiff.DecodeFile()` which attempts to read _only_ the metadata.
This is helpful when attempting to read hundreds and thousands of 100MB TIFF files. `DecodeFile`
works exactly the same as `tiff.Decode()` with the following caveats:

1. It assumes all metadata is stored after the first IFD entry
1. Because of the above it cannot deal with cases where some metadata value is stored _before_ the
   start of the first IFD entry

goexif
======

Provides decoding of basic exif and tiff encoded data. Still in alpha - no guarantees.
Suggestions and pull requests are welcome.  Functionality is split into two packages - "exif" and "tiff"
The exif package depends on the tiff package. 
Documentation can be found at http://godoc.org/github.com/rwcarlsen/goexif

Like goexif? - Bitcoin tips welcome: 17w65FVqx196Qp7tfCCSLqyvsHUhiEEa7P

To install, in a terminal type:

```
go get github.com/rwcarlsen/goexif/exif
```

Or if you just want the tiff package:

```
go get github.com/rwcarlsen/goexif/tiff
```

Example usage:

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
)

func ExampleDecode() {
	fname := "sample1.jpg"

	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}

	// Optionally register camera makenote data parsing - currently Nikon and
	// Canon are supported.
	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	camModel, _ := x.Get(exif.Model) // normally, don't ignore errors!
	fmt.Println(camModel.StringVal())

	focal, _ := x.Get(exif.FocalLength)
	numer, denom, _ := focal.Rat2(0) // retrieve first (only) rat. value
	fmt.Printf("%v/%v", numer, denom)

	// Two convenience functions exist for date/time taken and GPS coords:
	tm, _ := x.DateTime()
	fmt.Println("Taken: ", tm)

	lat, long, _ := x.LatLong()
	fmt.Println("lat, long: ", lat, ", ", long)
}
```

<!--golang-->
[![githalytics.com alpha](https://cruel-carlota.pagodabox.com/5e166f74cdb82b999ccd84e3c4dc4348 "githalytics.com")](http://githalytics.com/rwcarlsen/goexif)
