#!/bin/bash

# script to remove package names from generated Swagger doc

docs_dir="docs"
package_names=$(find . -name "*.go" -exec grep -h "^package " {} \; | awk '{print $2}' | sort | uniq)

for file in "$docs_dir"/*; do
    for package in $package_names; do
        sed -i "s/${package}\.//g" "$file"
    done
done
