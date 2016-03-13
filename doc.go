// Package coquery is an experiement in providing a different way of access and
// querying underline database systems with an approach that gives the client
// more control of what they want from the backend than the formal prescribed
// approach of defined endpoints(Restful CRUD) that only define routes for
// such operations. We are entering into grounds where exact requirements of
// parts of the UI are differing and allowing clients to only receive what matters
// to them as become as critical as any part of our application behaviour.
// eg
/*

// Retrieve all records and collect only the "name","age" and "address" properties.
docs.users.findN(-1).collects(name,age,address)

// Retrieve 10 records and collect only the "name","age" and "address" properties.
docs.users.findN(10).collects(name,age,address)

// Retrieve 10 records for the first set and collect only the "name","age" and "address" properties.
docs.users.findN(10,1).collects(name,age,address)

// Retrieve 10 records for the next set after the first and collect only the "name","age" and "address" properties.
docs.users.findN(10,2).collects(name,age,address)

// Retrieve record with the id=10 and collect only the "name","age" and "address" properties.
docs.users.find(id,10).collects(name,age,address)


// Retrieve record with the id=10 and mutate the "name" property to alex.
docs.user.find(id,0).mutate({name:"alex"})

*/
package coquery
