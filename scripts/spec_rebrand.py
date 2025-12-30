import os
import re

def rebrand():
    replacements = [
        (r'(?i)AAP-?0?111', 'AAP-001'),
        (r'(?i)AAP-?0?115', 'AAP-002'),
        (r'(?i)RFC-?0?111', 'AAP-001'),
        (r'(?i)RFC-?0?115', 'AAP-002'),
        (r'aap_0111', 'aap_001'),
        (r'aap_0115', 'aap_002'),
        (r'AAP-0111-compliant', 'AAP-001-compliant'),
        (r'ComplianceLevel: "AAP-0111-compliant"', 'ComplianceLevel: "AAP-001-compliant"'),
    ]

    target_dirs = ['docs', 'pkg', 'test', 'web', 'examples', 'cmd', 'internal']
    
    for target_dir in target_dirs:
        for root, dirs, files in os.walk(target_dir):
            for file in files:
                if file.endswith(('.go', '.md', '.json', '.html', '.txt', '.yaml', '.yml')):
                    path = os.path.join(root, file)
                    with open(path, 'r', encoding='utf-8', errors='ignore') as f:
                        content = f.read()
                    
                    new_content = content
                    for pattern, subst in replacements:
                        new_content = re.sub(pattern, subst, new_content)
                    
                    if new_content != content:
                        print(f"Updating {path}")
                        with open(path, 'w', encoding='utf-8') as f:
                            f.write(new_content)

if __name__ == "__main__":
    rebrand()
