#!/bin/bash

exec curl -i -H "Content-Type: application/json" -X POST http://localhost:50000/api --data '{"firstname":"John","lastname":"Doe"}'
