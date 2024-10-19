#!/bin/zsh

hey -n 100 -c 2 -m POST \
  -A "application/json" \
  -H "Content-Type: application/json" \
  -d '{"ticket_name":"test1","ticket_amount":2,"attendees":[{"a":"a"},{"b":"b"}]}' \
  'http://localhost:4000/v1/events/6da0a770-705f-4ff7-bd44-bf61181366dd/tickets/buy'