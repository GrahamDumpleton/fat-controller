load("@ytt:struct", "struct")
load("@ytt:regexp", "regexp")

load("random.star", "random")

_lower = "abcdefghijklmnopqrstuvwxyz"
_upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
_alpha = _lower + _upper
_digit = "0123456789"
_alnum = _alpha + _digit
_punct = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
_graph = _alnum + _punct

_xdigit = "0123456789ABCDEFabcdef"
_passwd = "!#%+23456789:=?@ABCDEFGHJKLMNPRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

patterns = {
  "lower": _lower,
  "upper": _upper,
  "alpha": _alpha,
  "digit": _digit,
  "alnum": _alnum,
  "punct": _punct,
  "graph": _graph,
  "xdigit": _xdigit,
  "passwd": _passwd
}

def _generate(value="[:passwd:]{16}"):
  found = True

  while found:
    found = False

    for name, charset in patterns.items():
      pattern = "\\[:%s:\\]{([0-9]+)}" % name
      if regexp.match(pattern, value):
        matches = regexp.replace("^.*?({}).*$".format(pattern), value, "$1 $2")
        if matches != value:
          found = True
          text, count = matches.split(" ")
          replacement = "".join([random.choice(charset) for _ in range(int(count))])
          value = value.replace(text, replacement, 1)
          break
        end
      end
    end
  end

  return value
end

password = struct.make(generate=_generate)
