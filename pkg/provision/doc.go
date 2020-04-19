// Package provision is a daemon that pulls
// on reservation source, and then tries to
// apply these reservations locally.
// Note that, provision module doesn't expose
// any interface on zbus. since it should not
// be driven by users, instead all reservation
// should be pushed by the reservation source.
package provision
