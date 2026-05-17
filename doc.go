// Package iploc parses QQWry (纯真 IP 库) IPv4 location databases.
//
// The package supports IPv4 only. IPv6 input returns ErrInvalidIP.
//
// A parser loads the database into memory, finds the matching IPv4 range with a
// binary search over the QQWry index, follows QQWry record redirects, and
// converts GBK record text to UTF-8.
//
// Use Query or QueryResult for new code because they report invalid IPv4 input
// and database corruption. Find is kept for compatibility with earlier versions
// and returns empty strings when Query would return an error.
package iploc
