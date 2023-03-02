load("@ytt:struct", "struct")
load("@ytt:regexp", "regexp")

load("functions.star", "xgetattr")

def _NameSelector__new(selector):
  def _NameSelector__match(name):
    match_names = xgetattr(selector, "matchNames", [])

    return name in match_names
  end

  self = struct.make(
    match=_NameSelector__match
  )

  return self
end

def _LabelSelector__new(selector):
  def _LabelSelector__match(resource):
    match_labels = xgetattr(selector, "matchLabels", {})
    match_expressions = xgetattr(selector, "matchExpressions", [])

    if not match_labels and not match_expressions:
      return True
    end
  
    labels = xgetattr(resource, "metadata.labels", struct.make())

    for key in match_labels:
      if xgetattr(labels, key) != match_labels[key]:
        return False
      end
    end

    for expression in match_expressions:
      key = expression.key
      operator = expression.operator
      values = xgetattr(expression, "values", [])
      value = xgetattr(labels, key)
      
      if operator == "In":
        if values:
          if value == None or value not in values:
            return False
          end
        end
      elif operator == "NotIn":
        if values:
          if value != None and value in values:
            return False
          end
        end
      elif operator == "Exists":
        if key not in labels:
          return False
        end
      elif operator == "DoesNotExist":
        if key in labels:
          return False
        end
      end
    end

    return True
  end

  self = struct.make(
    match=_LabelSelector__match
  )

  return self
end

selectors = struct.make(
  NameSelector=_NameSelector__new,
  LabelSelector=_LabelSelector__new)
