import json
import urllib.request

url = "https://repoe-fork.github.io/base_items.json"

with urllib.request.urlopen(url) as response:
    data = json.load(response)

result = {v["name"]: v["item_class"] for v in data.values()}

with open("item_classes.json", "w") as f:
    json.dump(result, f, indent=2)
