package swarm

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestCleanGRPCPeers verifies peer addresses are explicit.
*/
func TestCleanGRPCPeers(t *testing.T) {
	Convey("Given peer addresses", t, func() {
		Convey("When a peer address is blank", func() {
			peers, err := cleanGRPCPeers([]string{"127.0.0.1:1", ""})

			Convey("Then validation should fail", func() {
				So(peers, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})
}
