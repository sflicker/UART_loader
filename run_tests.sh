#!/bin/bash

test_dir="../SAP-2_System/asm_test_files"

go_program="./fileloader"

pass=0
fail=0
failed_files=()

for file in "$test_dir"/test*.asm; do
  echo "$file"
  "$go_program" "$file"

  exit_code=$?

  if [ $exit_code -eq 0 ]; then
    echo "Test $file passed"
    pass=$((pass + 1))
  else
    echo "Test $file failed"
    fail=$((fail + 1))
    failed_files+=("$file")
  fi
done

echo "Tests Completed: $pass passed, $fail failed"

if [ $fail -gt 0 ] ; then
  echo "Failed tests:"
  for file in "$failed_files[0]]}"; do
     echo "  $file"
  done
  exit 1
else
  exit 0
fi

