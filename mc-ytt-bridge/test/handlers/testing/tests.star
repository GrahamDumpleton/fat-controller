load("@ytt:data", "data")

def echo(**kwargs):
  return kwargs
end

def values(**_):
  return data.values
end

def files(**_):
  return {"files": data.list("/")}
end

def random(**_):
  return {"seed": hash(data.read("/__random__.dat"))}
end
