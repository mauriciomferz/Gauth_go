bg
import sys
import re

def normalize_file(filepath):
    with open(filepath, 'r') as f:
        lines = f.readlines()

    new_lines = []
    in_block = False
    
    # Regex to match code block fence
    fence_pattern = re.compile(r'^```\s*(\w*)\s*$')

    count_fixed = 0
    count_tagged = 0

    for line in lines:
        stripped = line.strip()
        match = fence_pattern.match(line)
        
        if match:
            # It's a fence
            tag = match.group(1)
            
            if not in_block:
                # Opening fence
                in_block = True
                if not tag:
                    # Untagged opening -> Tag as text
                    new_lines.append("```text\n")
                    count_fixed += 1
                else:
                    new_lines.append(line)
                    count_tagged += 1
            else:
                # Closing fence
                in_block = False
                new_lines.append("```\n") # Always clean closing
        else:
            new_lines.append(line)

    with open(filepath, 'w') as f:
        f.writelines(new_lines)

    print(f"Fixed {count_fixed} untagged blocks. Found {count_tagged} already tagged blocks.")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python normalize.py <file>")
        sys.exit(1)
    normalize_file(sys.argv[1])
