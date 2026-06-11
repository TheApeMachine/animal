package lease

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestPrincipalFromIdentity builds principals from actor identities.
*/
func TestPrincipalFromIdentity(t *testing.T) {
	Convey("Given a nil identity", t, func() {
		Convey("When PrincipalFromIdentity is called", func() {
			principal := PrincipalFromIdentity(nil)

			Convey("Then it should return an empty principal", func() {
				So(principal.ActorID, ShouldBeEmpty)
			})
		})
	})
}
