// Package coquery provides a makeshift query system that provides an interesting
// view to how data is to be collected and mutated.
// eg
/*

  docs.user.rid(4356932).kv(id,0).keys(name,age,address)

  kv => {type: find, key:id, value: 0}
  keys => {type: collect, keys: [name, age ,address]}

  docs.user.kv(id,0).mutate({name:"alex"})

  kv => {type: find, key:id, value: 0}
  mutate => {type: mutate, keys: {name: "alex"}}
*/
package coquery
