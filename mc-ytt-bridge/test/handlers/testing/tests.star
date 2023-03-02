load("@ytt:data", "data")

def echo(**kwargs):
  return kwargs
end

def values(**_):
  return data.values
end
