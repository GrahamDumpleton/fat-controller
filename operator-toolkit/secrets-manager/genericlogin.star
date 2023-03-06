load("@ytt:base64", "base64")

load("/functions.star", "xgetattr")
load("/random.star", "random")

load("genericlogin.lib.yaml", "new_generic_login_secret")

# Applies rules of the custom resource against resources we are interested in.

def sync(parent, children, **_):
  resources = []

  existing = xgetattr(children["Secret.v1"], parent.metadata.name)

  data = {}

  if existing == None:
    data["username"] = base64.encode(parent.spec.credentials.username)
    data["password"] = base64.encode(generate_password(parent))
  else:
    data = xgetattr(existing, "data", {})
  end

  resources = [new_generic_login_secret(parent, parent.metadata.name, parent.metadata.namespace, data)]

  return { "children": resources }
end

_lower = "abcdefghijklmnopqrstuvwxyz"
_upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
_alpha = _lower + _upper
_digit = "0123456789"
_alnum = _alpha + _digit
_punct = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
_graph = _alnum + _punct

def generate_password(parent):
  length = int(xgetattr(parent, "spec.credentials.password.length", 32))

  digits_minimum = int(xgetattr(parent, "spec.credentials.password.digits.minimum", 0))
  lowercase_minimum = int(xgetattr(parent, "spec.credentials.password.lowercase.minimum", 0))
  uppercase_minimum = int(xgetattr(parent, "spec.credentials.password.uppercase.minimum", 0))
  symbols_minimum = int(xgetattr(parent, "spec.credentials.password.symbols.minimum", 0))

  symbols_charset = []

  for c in xgetattr(parent, "spec.credentials.password.symbols.charset", _punct).elems():
    if c in _punct:
      symbols_charset.append(c)
    end
  end

  characters = []

  for _ in range(digits_minimum):
    characters.append(random.choice(_digit))
  end

  for _ in range(lowercase_minimum):
    characters.append(random.choice(_lower))
  end

  for _ in range(uppercase_minimum):
    characters.append(random.choice(_upper))
  end

  for _ in range(symbols_minimum):
    characters.append(random.choice(symbols_charset))
  end

  string_charset = _alnum + "".join(symbols_charset)

  while len(characters) < length:
    characters.append(random.choice(string_charset))
  end

  password = []

  for _ in range(length):
    password.append(characters.pop(random.randint()%len(characters)))
  end

  return "".join(password)
end
