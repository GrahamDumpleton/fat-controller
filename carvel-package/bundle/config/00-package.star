load("@ytt:data", "data")
load("@ytt:base64", "base64")
load("@ytt:json", "json")

def image_reference(name):
  registry = data.values.imageRegistry.host
  if not registry:
    registry = "registry.default.svc.cluster.local:5001"
  end
  if data.values.imageRegistry.namespace:
    registry = "{}/{}".format(registry, data.values.imageRegistry.namespace)
  end
  image = "{}/fat-controller/{}:{}".format(registry, name, data.values.version)
  for item in data.values.imageVersions:
    if item.name == name:
      image = item.image
      break
    end
  end
  return image
end

def image_pull_policy(image):
  tag = image.split(":")
  always = len(tag) <= 1 or tag[-1] in ["latest", "main", "master", "develop"]
  return always and "Always" or "IfNotPresent"
end
