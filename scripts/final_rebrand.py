import os
import re

REPLACEMENTS = [
    (re.compile(r'RFC-0111', re.IGNORECASE), 'AAP-001'),
    (re.compile(r'RFC-0115', re.IGNORECASE), 'AAP-002'),
    (re.compile(r'GAUTH', re.IGNORECASE), 'AGENTAUTH'),
    (re.compile(r'Gauth', re.IGNORECASE), 'AgentAuth'),
    (re.compile(r'gauth', re.IGNORECASE), 'agentauth'),
    (re.compile(r'Siemens', re.IGNORECASE), 'AgentAuth'),
    (re.compile(r'Gimel', re.IGNORECASE), 'AgentAuth'),
    (re.compile(r'Amtsgericht München', re.IGNORECASE), 'Central Registry'),
    (re.compile(r'Amtsgericht Berlin', re.IGNORECASE), 'District Registry'),
]

TARGET_DIRS = ['pkg', 'internal', 'web', 'cmd', 'examples', 'conformance']

def process_file(file_path):
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
    except UnicodeDecodeError:
        return # Skip binary files

    new_content = content
    for pattern, replacement in REPLACEMENTS:
        # Special case for cases where we already have AgentAuth (don't double up)
        # But simple regex sub is usually fine if patterns are distinct
        new_content = pattern.sub(replacement, new_content)

    if new_content != content:
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(new_content)
        print(f"Updated: {file_path}")

def main():
    for target in TARGET_DIRS:
        abs_path = os.path.join(os.getcwd(), target)
        if not os.path.exists(abs_path):
            continue
        for root, dirs, files in os.walk(abs_path):
            for file in files:
                if file.endswith(('.go', '.md', '.json', '.yaml', '.yml', '.html', '.css', '.js', '.ts', '.tsx')):
                    process_file(os.path.join(root, file))

if __name__ == "__main__":
    main()
