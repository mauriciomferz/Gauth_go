#!/usr/bin/env python3
import re
import csv
import os

def parse_gap_matrix(md_path):
    with open(md_path, 'r') as f:
        content = f.read()

    # Find sections like "## 1. Cryptographic & Authenticity"
    sections = re.findall(r'## \d+\. (.*?)\n(.*?)(?=\n## \d+|\Z)', content, re.DOTALL)
    
    rows = []
    
    for section_name, section_content in sections:
        section_name = section_name.strip()
        
        # Find tables in this section
        # | Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
        # Note: The structure in GAP_MATRIX.md sometimes differs between sections. 
        # Section 1-11 seem to follow a standard table format.
        
        table_match = re.search(r'\| Requirement \|.*?\|\n\|[-| ]+\|\n(.*?)(?=\n\n|\Z)', section_content, re.DOTALL)
        if not table_match:
            continue
            
        table_rows = table_match.group(1).strip().split('\n')
        for row_str in table_rows:
            cols = [c.strip() for c in row_str.split('|')]
            if len(cols) < 8:
                continue
            
            # Typical columns: 
            # , Requirement, Current Implementation, Gap, Status, Priority, Impact, Suggested Action, 
            # We want: Section, ID, Requirement, Status, Priority, Gap, Evidence (Evidence will be empty or Current Implementation)
            
            requirement = cols[1]
            current_impl = cols[2]
            gap = cols[3]
            status = cols[4]
            priority = cols[5]
            
            # Map Requirement to ID if possible? The original CSV had IDs like "sec1.item1".
            # Mapping is hard without ID in the table. 
            # However, we can use a simple counter or just keep it as is.
            # Original CSV ID was secX.itemY.
            
            rows.append({
                'Section': section_name,
                'ID': 'N/A', # Placeholder as ID is not in MD table
                'Requirement': requirement,
                'Status': status,
                'Priority': priority,
                'Gap': gap,
                'Evidence': current_impl
            })
            
    return rows

def write_csv(rows, csv_path):
    fieldnames = ['Section', 'ID', 'Requirement', 'Status', 'Priority', 'Gap', 'Evidence']
    with open(csv_path, 'w', newline='') as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for row in rows:
            writer.writerow(row)

if __name__ == "__main__":
    md_file = "docs/GAP_MATRIX.md"
    csv_file = "artifacts/gap_matrix.csv"
    
    if os.path.exists(md_file):
        print(f"Parsing {md_file}...")
        rows = parse_gap_matrix(md_file)
        print(f"Found {len(rows)} requirements.")
        write_csv(rows, csv_file)
        print(f"Successfully updated {csv_file}")
    else:
        print(f"Error: {md_file} not found.")
