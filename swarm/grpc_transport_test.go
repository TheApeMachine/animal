package swarm

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestNewGRPCMeshTransport verifies loopback gRPC transport construction.
*/
func TestNewGRPCMeshTransport(t *testing.T) {
	Convey("Given a loopback gRPC listener address", t, func() {
		ctx := context.Background()

		Convey("When a transport is created", func() {
			transport, err := NewGRPCMeshTransport(ctx, "127.0.0.1:0", nil)
			So(err, ShouldBeNil)

			defer func() {
				So(transport.Close(), ShouldBeNil)
			}()

			Convey("Then it should expose the bound address", func() {
				So(transport.Address(), ShouldNotEqual, "")
			})
		})
	})
}
