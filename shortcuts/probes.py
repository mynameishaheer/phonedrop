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


PROBES = [
    probe("1 select photos", lambda s: s.act("selectphoto", {"WFSelectMultiplePhotos": True}, "Photos")),
    probe("2 if shortcut input", lambda s: (s.if_(b.shortcut_input(), b.HAS_VALUE), s.alert("x"), s.end()),
          share_sheet=True, input_classes=["WFImageContentItem", "WFGenericFileContentItem"]),
    probe("3 count and number if", lambda s: (
        s.set("C", s.act("count", {"Input": b.attach(b.var("T")), "WFCountType": "Items"}, "Count")),
        s.if_(b.var("C"), b.IS, number="0"), s.alert("x"), s.end())),
    probe("4 copy to clipboard", lambda s: s.act("setclipboard", {"WFInput": b.attach(b.var("T"))})),
    probe("5 open urls", lambda s: s.act("openurl", {"WFInput": b.attach(b.var("T"))})),
    probe("6 save to photos", lambda s: s.act("savetocameraroll", {"WFInput": b.attach(b.var("T")),
                                                                    "WFCameraRollSelectedGroup": "Recents"})),
    probe("7 save file ask", lambda s: s.act("documentpicker.save", {"WFInput": b.attach(b.var("T")),
                                                                      "WFAskWhereToSave": True}, "Saved File")),
    probe("8 dictionary key of repeat item", lambda s: (
        s.repeat_each(b.var("T")), s.act("setclipboard", {"WFInput": b.attach(b.var(b.REPEAT_ITEM, "text"))}),
        s.end())),
]

if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    for s in PROBES:
        with open(os.path.join(OUT, s.name + ".shortcut"), "wb") as fh:
            plistlib.dump(s.workflow(), fh, fmt=plistlib.FMT_BINARY, sort_keys=True)
    print("\n".join(sorted(os.listdir(OUT))))
