// Package coquery is an experiement in providing a different way of access and
// querying underline database systems with an approach that gives the client
// more control of what they want from the backend than the formal prescribed
// approach of defined endpoints(Restful CRUD) that only define routes for
// such operations. We are entering into grounds where exact requirements of
// parts of the UI are differing and allowing clients to only receive what matters
// to them as become as critical as any part of our application behaviour.
// eg
/*

  docs.user.rid(4356932).find(id,10).collects(name,age,address)

  find => {type: find, key:id, value: 0}
  collects => {type: collect, keys: [name, age ,address]}

  docs.user.find(id,0).mutate({name:"alex"})

  find => {type: find, key:id, value: 0}
  mutate => {type: mutate, keys: {name: "alex"}}
*/
package coquery
