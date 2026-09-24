#!/bin/sh
# Aplica as migrations pendentes e sobe a API. Se a migration falhar, a API não sobe.
set -e
./migrate up
exec ./api
