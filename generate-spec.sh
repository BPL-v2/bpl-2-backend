#!/bin/zsh

# script to remove package names from generated Swagger doc
swag init
docs_dir="docs"
package_names=$(find . -name "*.go" -exec grep -h "^package " {} \; | awk '{print $2}' | sort | uniq)

for file in "$docs_dir"/*; do
    # Convert package_names to array for proper iteration in zsh
    package_array=(${=package_names})
    for package in $package_array; do
        # Use backup suffix for macOS compatibility
        sed -i.bak "s/${package}\.//g" "$file" && rm "${file}.bak"
    done
done
