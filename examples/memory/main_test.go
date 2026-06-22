package main

import (
	"bytes"
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestNewDemo verifies memory demo construction.
*/
func TestNewDemo(t *testing.T) {
	Convey("Given a context and output writer", t, func() {
		var output bytes.Buffer

		demo, err := NewDemo(context.Background(), &output)

		Convey("It should create a demo", func() {
			So(err, ShouldBeNil)
			So(demo, ShouldNotBeNil)
		})
	})
}
