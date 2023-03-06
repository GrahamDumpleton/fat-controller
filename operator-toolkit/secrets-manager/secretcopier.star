load("/functions.star", "xgetattr")
load("/selectors.star", "selectors")

load("secretcopier.lib.yaml", "generate_secret")

# Returns rules for metacontroller that match resources we are interested in for
# the supplied custom resource instance.

def customize(parent, **_):
  rules = []

  # A single secret copier for convenience can contain multiple rules so we need
  # to check all of them.

  for rule in parent.spec.rules:
    # First construct rule for metacontroller matching on source secret.

    rules.append(
      {
        "apiVersion": "v1",
        "resource": "secrets",
        "namespace": rule.sourceSecret.namespace,
        "names": [rule.sourceSecret.name]
      }
    )

    # Next we need to construct a rule for metacontroller which matches the
    # target namespaces. Note that there is no way to target all namespaces via
    # a wildcard. If you want to match all namespaces you would need to use a
    # dummy label selector which would match all namespaces.

    match_names = xgetattr(rule, "targetNamespaces.nameSelector.matchNames")
    label_selector = xgetattr(rule, "targetNamespaces.nameSelector.labelSelector")

    if match_names:
      rules.append(
        {
          "apiVersion": "v1",
          "resource": "namespaces",
          "names": match_names
        }
      )
    elif label_selector:
      rules.append(
        {
          "apiVersion": "v1",
          "resource": "namespaces",
          "labelSelector": label_selector
        }
      )
    end
  end

  return { "relatedResources": rules }
end

# Applies rules of the custom resource against resources we are interested in.

def sync(parent, related, **_):
  # If there are no source secrets we return an empty list, which if there were
  # previously secrets which had been copied, will result in the copied secrets
  # being deleted from the target namespaces.

  if len(related["Secret.v1"]) == 0:
    return { "children": [] }
  end

  # A single secret copier for convenience can contain multiple rules so we need
  # to check all of them.

  resources = []

  for rule in parent.spec.rules:
    resources.extend(process_rule(parent, related, rule))
  end

  # Return the list of secret resources which should exist based on the rules.
  # Note that we aren't doing anything to deal with the case where multiple
  # rules within the one custom resource could result in the same named secret
  # being created which didn't exist before. In this case the last would take
  # priority since we are doing an in place update and we are the owner, so the
  # earlier version will be overridden.

  return { "children": resources }
end

# Process a single rule and generate the list of secrets that need to exist.

def process_rule(parent, related, rule):
  # See if there is a source secret matching the rule. If there isn't we can
  # bail out immediately.

  source_secret_namespace = rule.sourceSecret.namespace
  source_secret_name = rule.sourceSecret.name
  source_secret_key = "{}/{}".format(source_secret_namespace, source_secret_name)

  source_secret = getattr(related["Secret.v1"], source_secret_key, None)

  if not source_secret:
    return []
  end

  # We now need to calculate the set of target namespaces matched by the rule.
  # It is required that there must be either a name selector or label selector.

  matches = []

  name_selector_rules = xgetattr(rule, "targetNamespaces.nameSelector", None)
  name_selector = selectors.NameSelector(name_selector_rules)

  label_selector_rules = xgetattr(rule, "targetNamespaces.labelSelector", None)
  label_selector = selectors.LabelSelector(label_selector_rules)

  for name in related["Namespace.v1"]:
    namespace = related["Namespace.v1"][name]
    matched = False

    if name_selector_rules != None:
      matched = name_selector.match(namespace.metadata.name)
    elif label_selector_rules != None:
      matched = label_selector.match(namespace)
    end

    if matched:
      matches.append((rule, namespace))
    end
  end

  # Finally generate the set of secrets that need to exist to satisfy the rule.
  # Note that we don't have to worry about whether conflicting custom resources
  # try and update the same target secret as metacontroller will not allow it as
  # the secret will already have a different owner.

  resources = []

  for rule, namespace in matches:
    target_secret_name = xgetattr(rule, "targetSecret.name", source_secret_name)
    resources.append(generate_secret(parent, rule, namespace, target_secret_name, source_secret))
  end

  return resources
end
