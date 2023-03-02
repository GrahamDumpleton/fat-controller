def xgetattr(object, path, default=None):
  def _lookup(object, key, default=None):
    keys = key.split(".")
    value = default
    for key in keys:
      value = getattr(object, key, None)
      if value == None:
        return default
      end
      object = value
    end
    return value
  end

  return _lookup(object, path, default)
end
