# queryit

- `queryit` is a CLI based SQL query executor with full support for most relational database flavors. Currently the development only concerns itself with postgres connections with 2 authentication methods:
  1. username / password : direct host connection with user and password
  2. ssh with bastion (jump host): ssh to a bastion with a PEM file and then query the db from there

- the CLI will have a persistent input box below with new line support and ability to change focus between the input box and the output

- all connection settings, along with configurations to be stored in an yaml or json file in local

- the goal is simplicity: ability to run SQL queries, list all tables (through commands), inspect schema, run edits;

- there must be configurable settings (like table rendering as an expanded list which is scrollable) or  markdown table render

- we can choose which language (go or rust) will give us the most flexibility in terms of the TUI

- flush out the entire spec and come up with a detailed plan of action

- extrapolate on the required features and ask as many clarifying questions as you want
