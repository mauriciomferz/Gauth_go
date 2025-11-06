#!/usr/bin/env python3
"""Generate CollectorRegistry dispatch methods from noop implementation."""

import re
import sys

def parse_noop_method(line):
    """Parse noop method signature: func (n noop) MethodName(params) {}
    
    Handles Go's parameter shorthand where multiple parameters share a type:
    - "a, b, c string" means all three are strings
    - "a string, b int" means a is string, b is int
    """
    match = re.match(r'func \(n noop\) (\w+)\(([^)]*)\)', line)
    if not match:
        return None
    
    method_name = match.group(1)
    params_str = match.group(2).strip()
    
    if not params_str:
        return method_name, [], []
    
    # Split by commas
    raw_parts = [p.strip() for p in params_str.split(',')]
    
    params = []
    param_names = []
    pending_names = []  # Names waiting for a type
    
    for part in raw_parts:
        tokens = part.split()
        
        if len(tokens) == 2:
            # "name type" - this completes any pending names
            name, typ = tokens
            
            # Add pending names with this type
            for pending in pending_names:
                params.append(f"{pending} {typ}")
                param_names.append(pending)
            pending_names = []
            
            # Add current parameter
            params.append(f"{name} {typ}")
            param_names.append(name)
            
        elif len(tokens) == 1:
            # Just a name (type comes later)
            pending_names.append(tokens[0])
        else:
            # Complex type or empty - skip
            pass
    
    return method_name, params, param_names

def generate_registry_method(method_name, params, param_names):
    """Generate registry dispatch method."""
    # Rename any parameter named 'r' to avoid conflict with receiver
    renamed_params = []
    renamed_param_names = []
    for i, (param, param_name) in enumerate(zip(params, param_names)):
        if param_name == 'r':
            # Rename to avoid conflict with receiver
            renamed_param = param.replace('r ', 'r2 ')
            renamed_params.append(renamed_param)
            renamed_param_names.append('r2')
        else:
            renamed_params.append(param)
            renamed_param_names.append(param_name)
    
    params_str = ", ".join(renamed_params)
    call_args = ", ".join(renamed_param_names)
    
    lines = [
        f"func (reg *CollectorRegistry) {method_name}({params_str}) {{",
        f"\treg.dispatch(func(c MetricsCollector) {{ c.{method_name}({call_args}) }})",
        "}",
        ""
    ]
    return "\n".join(lines)

def main():
    # Read metrics.go
    with open('/Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/internal/metrics/metrics.go', 'r') as f:
        content = f.read()
    
    # Find all noop methods
    noop_methods = []
    for line in content.split('\n'):
        if line.startswith('func (n noop)'):
            parsed = parse_noop_method(line)
            if parsed:
                noop_methods.append(parsed)
    
    print(f"// Auto-generated registry dispatch methods ({len(noop_methods)} total)")
    print()
    
    # Generate registry methods
    for method_name, params, param_names in noop_methods:
        print(generate_registry_method(method_name, params, param_names))

if __name__ == '__main__':
    main()
