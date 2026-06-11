package config

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/lease"
)

/*
TestLeaseSectionIdleTTL verifies lease idle duration from config values.
*/
func TestLeaseSectionIdleTTL(t *testing.T) {
	Convey("Given a positive idle_ttl_seconds value", t, func() {
		section := LeaseSection{IdleTTLSeconds: 900}

		Convey("When IdleTTL is called", func() {
			idleTTL, err := section.IdleTTL()

			Convey("Then it should return the configured duration", func() {
				So(err, ShouldBeNil)
				So(idleTTL, ShouldEqual, 900*time.Second)
			})
		})
	})

	Convey("Given a missing idle_ttl_seconds value", t, func() {
		section := LeaseSection{}

		Convey("When IdleTTL is called", func() {
			_, err := section.IdleTTL()

			Convey("Then it should reject the missing configuration", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

/*
TestLeaseSectionCoordinatorOptions verifies lease.Options construction.
*/
func TestLeaseSectionCoordinatorOptions(t *testing.T) {
	Convey("Given lease defaults and a key space", t, func() {
		section := LeaseSection{IdleTTLSeconds: 120}

		Convey("When CoordinatorOptions is called", func() {
			options, err := section.CoordinatorOptions(lease.PathKeySpace{})

			Convey("Then it should return coordinator options from config", func() {
				So(err, ShouldBeNil)
				So(options.IdleTTL, ShouldEqual, 120*time.Second)
			})
		})
	})
}
