"""Tiny one-action shortcuts for finding which action Apple's signing server rejects.

python3 -I probes.py  ->  probes/unsigned/*.shortcut; sign them with sign-all.sh's logic
(bash sign-probes.sh on the Mac). Whichever probe fails points at the culprit.
"""
import os
import plistlib
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # works under python3 -I too
import build as b  # noqa: E402

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "probes", "unsigned")


def probe(name, fn, **kw):
    s = b.Shortcut("Probe " + name, **kw)
    s.set("T", s.text("hello"))
    fn(s)
    return s


def _p2a(s):
    s.set("N", s.act("count", {"Input": b.attach(b.shortcut_input()), "WFCountType": "Items"}, "Count"))
    s.if_(b.var("N"), b.IS, number="0")
    s.alert("x")
    s.end()


def _p5a(s):
    u = s.act("url", {"WFURLActionURL": b.tok(b.var("T"))}, "URL")
    s.act("openurl", {"WFInput": b.attach(u)})


def _p8a(s):
    d = s.dictionary([("items", "x")])
    s.repeat_each(s.get_value(d, "items"))
    s.set("V", s.get_value(b.var(b.REPEAT_ITEM), "text"))
    s.act("setclipboard", {"WFInput": b.attach(b.var("V"))})
    s.end()


# Round 2: the alternatives used by the rewritten Send to PC and Receive from PC.
PROBES = [
    probe("2a count shortcut input", _p2a, share_sheet=True,
          input_classes=["WFImageContentItem", "WFGenericFileContentItem"]),
    probe("5a url then open urls", _p5a),
    probe("8a get value of repeat item", _p8a),
]

if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    for s in PROBES:
        with open(os.path.join(OUT, s.name + ".shortcut"), "wb") as fh:
            plistlib.dump(s.workflow(), fh, fmt=plistlib.FMT_BINARY, sort_keys=True)
    print("\n".join(sorted(os.listdir(OUT))))
