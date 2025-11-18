#!/usr/bin/env zsh

for d in ${(s/:/)PATH}; do
    if [ -w "$d" ]; then
        echo "$d  (writable)"
    else
        echo "$d"
    fi
done
