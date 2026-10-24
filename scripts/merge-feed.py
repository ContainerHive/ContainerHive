#!/usr/bin/env python3
"""Merges a 0template-generated feed into the 0install master feed and signs it.

Usage: merge-feed.py [--init FEED_URI] [--canonical FEED_URI] <master-feed> <generated-feed> <gpg-key-id>

--init creates the master feed from the generated one and sets the canonical
feed URI. Without --init the implementations of the generated feed are merged
into the matching per-arch groups. Implementations with the same version are
replaced, so a re-run is safe.

--canonical re-points the feed-for element and the deprecated root uri
attribute of the result to the given feed URI. Pass it on every run, so a
canonical feed URL change survives the next merge.

The result is signed with a detached GPG signature embedded as a Base64
comment block, the same format 0install clients accept. Requires gpg on
PATH with the imported private key.
"""

import base64
import os
import subprocess
import sys
import tempfile
from xml.dom import minidom

IFACE_NS = "http://zero-install.sourceforge.net/2004/injector/interface"
ELEMENT_NODE = 1
SIGNATURE_START = "<!-- Base64 Signature\n"
SIGNATURE_END = "\n-->\n"


def fail(message):
    print(f"error: {message}", file=sys.stderr)
    sys.exit(1)


def direct_children(node, tag):
    return [n for n in node.childNodes if n.nodeType == ELEMENT_NODE and n.localName == tag]


def run_command_paths(group):
    """Returns the path attributes of the group's own <command name="run"> children."""
    return [
        cmd.getAttribute("path")
        for cmd in direct_children(group, "command")
        if cmd.getAttribute("name") == "run"
    ]


def impl_keys(group):
    return [(imp.getAttribute("version"), imp.getAttribute("arch")) for imp in direct_children(group, "implementation")]


def group_index(interface):
    """Maps each top-level group to its run-command paths."""
    index = {}
    for group in direct_children(interface, "group"):
        for path in run_command_paths(group):
            index.setdefault(path, group)
    return index


def strip_signature_block(data):
    position = data.rfind(b"\n" + SIGNATURE_START.encode())
    if position != -1:
        return data[: position + 1]
    return data


def serialize(node):
    """Writes the document with one child element per line, text inline."""
    lines = []
    write_node(node, 0, lines)
    return "".join(lines)


def write_node(node, depth, lines):
    cap = "  " * depth
    tag = node.tagName
    attrs = "".join(f' {name}="{node.getAttribute(name)}"' for name in sorted_attrs(node))
    children = [n for n in node.childNodes if n.nodeType == ELEMENT_NODE]
    text = "".join(n.toxml() for n in node.childNodes if n.nodeType == n.TEXT_NODE)
    if children:
        lines.append(f"{cap}<{tag}{attrs}>\n")
        for child in children:
            write_node(child, depth + 1, lines)
        lines.append(f"{cap}</{tag}>\n")
    elif node.hasChildNodes():  # text only
        lines.append(f"{cap}<{tag}{attrs}>{text}</{tag}>\n")
    else:
        lines.append(f"{cap}<{tag}{attrs}/>\n")


def sorted_attrs(node):
    return sorted(node.attributes.keys())


def merge(master_path, generated_path, init_uri, canonical_uri):
    master_doc = minidom.parse(master_path) if init_uri is None else minidom.parse(generated_path)
    generated_doc = minidom.parse(generated_path)

    master_root = master_doc.documentElement
    generated_root = generated_doc.documentElement

    if init_uri is not None:
        # The generated feed becomes the master feed; set the canonical feed URI.
        master_root.setAttribute("uri", init_uri)

    if canonical_uri is not None:
        # Keep the canonical feed URL current, so a domain migration applies
        # to the published master feed with the next merge.
        if master_root.hasAttribute("uri"):
            master_root.setAttribute("uri", canonical_uri)
        for feed_for in direct_children(master_root, "feed-for"):
            feed_for.setAttribute("interface", canonical_uri)

    if init_uri is not None:
        return master_doc

    master_groups = group_index(master_root)
    for generated_group in direct_children(generated_root, "group"):
        paths = run_command_paths(generated_group)
        target = None
        for path in paths:
            if path in master_groups:
                target = master_groups[path]
                break
        if target is None:
            # Unknown platform layout: add the whole group verbatim.
            target = master_root.appendChild(master_doc.importNode(generated_group, True))
            continue
        known_keys = impl_keys(target)
        for impl in direct_children(generated_group, "implementation"):
            key = (impl.getAttribute("version"), impl.getAttribute("arch"))
            if key in known_keys:
                print(f"replacing existing implementation for version {key[0]} on {key[1]}")
                for existing in list(direct_children(target, "implementation")):
                    existing_key = (existing.getAttribute("version"), existing.getAttribute("arch"))
                    if existing_key == key:
                        target.removeChild(existing)
            target.appendChild(master_doc.importNode(impl, True))
    return master_doc


def sign(master_doc, key_id, gpg="gpg"):
    xml = b'<?xml version="1.0"?>\n' + serialize(master_doc.documentElement).encode("utf-8")
    data = strip_signature_block(xml) if xml.find(SIGNATURE_START.encode()) != -1 else xml

    with tempfile.NamedTemporaryFile(suffix=".feed", delete=False) as body_file:
        body_file.write(data)
        body_path = body_file.name
    sig_path = body_path + ".sig"
    subprocess.run(
        [gpg, "--batch", "--detach-sign", "--default-key", key_id, "--output", sig_path, body_path],
        check=True,
    )
    with open(sig_path, "rb") as file:
        encoded = base64.b64encode(file.read()).decode("ascii")
    os.unlink(body_path)
    os.unlink(sig_path)

    wrapped = "\n".join(encoded[i : i + 64] for i in range(0, len(encoded), 64))
    if not data.endswith(b"\n"):
        data += b"\n"
    return data + SIGNATURE_START.encode() + wrapped.encode("ascii") + SIGNATURE_END.encode()


def main():
    args = sys.argv[1:]
    init_uri = None
    canonical_uri = None
    while args[:1] == ["--init"] or args[:1] == ["--canonical"]:
        flag, value, args = args[0], args[1], args[2:]
        if flag == "--init":
            init_uri = value
        else:
            canonical_uri = value
    if len(args) != 3:
        print(__doc__)
        sys.exit(1)
    master_path, generated_path, key_id = args

    if init_uri is not None:
        result = merge(master_path, generated_path, init_uri=init_uri, canonical_uri=canonical_uri)
    else:
        if not os.path.exists(master_path):
            fail(f"master feed {master_path} does not exist")
        result = merge(master_path, generated_path, init_uri=None, canonical_uri=canonical_uri)

    with open(master_path, "wb") as file:
        file.write(sign(result, key_id))
    print(f"signed master feed written to {master_path}")


if __name__ == "__main__":
    main()
