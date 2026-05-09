import json
from collections import Counter
import re

with open('graphify-out/.graphify_analysis.json') as f:
    data = json.load(f)

labels = {}
for cid, nodes in data['communities'].items():
    words = []
    for node in nodes:
        parts = node.split('_')
        for part in parts:
            if len(part) > 2 and part not in ['domain', 'infrastructure', 'services', 'repository', 'handlers', 'go', 'new', 'get', 'handle']:
                words.append(part)
    
    if words:
        common = Counter(words).most_common(2)
        label = ' '.join([c[0].capitalize() for c in common])
        labels[int(cid)] = label + " Components"
    else:
        labels[int(cid)] = "Core Components"

print(repr(labels))
