load("@ytt:library", "library")

def sync(parent, **_):
  registry = library.get("registry")
  registry = registry.with_data_values(parent, plain=True)
  return { "children": [resource for resource in registry.eval()] }
end
