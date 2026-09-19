// Package base84 implements canonical Base84 binary-to-text encoding.
//
// Encode and Decode use the standard Base84 alphabet. NewEncoding creates an
// immutable encoding with a custom alphabet. NewEncoder and NewDecoder process
// streams incrementally. Decoding is strict and rejects invalid characters and
// non-canonical padding.
package base84
