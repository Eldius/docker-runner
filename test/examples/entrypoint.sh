#!/bin/ash

while true; do sleep 15 ; echo "background"; done &

while true; do sleep 12 ; echo "foreground"; done
