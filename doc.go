// Package coquery provides a makeshift query system that provides an interesting
// view to how data is to be collected and mutated.
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
