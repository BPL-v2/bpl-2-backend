#!/bin/bash

# script to remove package names from generated Swagger doc
swag init
docs_dir="docs"
package_names=$(find . -name "*.go" -exec grep -h "^package " {} \; | awk '{print $2}' | sort | uniq)

for file in "$docs_dir"/*; do
    for package in $package_names; do
        # Use portable sed for Linux and macOS
        if sed --version >/dev/null 2>&1; then
            # GNU sed (Linux)
            sed -i "s/${package}\.//g" "$file"
        else
            # BSD sed (macOS)
            sed -i '' "s/${package}\.//g" "$file"
        fi
    done
done