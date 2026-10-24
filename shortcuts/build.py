#!/usr/bin/env python3
"""Generate the PhoneDrop iPhone shortcuts as unsigned WFWorkflow plists.

    python3 -I shortcuts/build.py            # build into shortcuts/unsigned/ and validate
    python3 -I shortcuts/build.py --check    # only validate what is in shortcuts/unsigned/
    python3 -I shortcuts/build.py --xml DIR  # also write XML copies (for reading/diffing)

The files follow docs/SHORTCUTS.md and docs/PROTOCOL.md. They must be signed on a Mac
(shortcuts/sign-all.sh) before an iPhone will import them.

Stdlib only. Output is deterministic: action UUIDs are derived from the shortcut name and
the action's position, so rebuilding without changes gives identical bytes.

Notes on choices that are not obvious:

* Named variables (Set Variable) are used for every value that is used later, like the
  hand-built versions that were verified on the device.
* "Get File from Folder" / "Save File" carry only a path (WFGetFilePath /
  WFFileDestinationPath) and no folder bookmark. A folder bookmark is device specific
  (iCloud item ids), and without one the actions use their default folder, the Shortcuts
  folder in iCloud Drive (iCloud Drive/Shortcuts/PhoneDrop/config.json, PROTOCOL.md §6).
  If an imported action shows an empty Folder, tap it and pick iCloud Drive -> Shortcuts.
* Every dictionary lookup on a variable is "coerce to Dictionary, then key" (what the editor
  writes when you pick Dictionary -> key on Shortcut Input), so it also works when the value
  arrives as JSON text or a file.
"""

import argparse
import os
import plistlib
import re
import sys
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))
OUT_DIR = os.path.join(HERE, "unsigned")

CLIENT_VERSION = "2607.0.2"
OBJ = "￼"  # object replacement character: where a variable sits in a text field
NS = uuid.UUID("6f1c2a52-7d0e-4c55-9a3e-0b9d6d2f5a10")

SIGN_NAME = "PhoneDrop Sign"

# WFCondition codes
IS, IS_NOT, BEGINS_WITH = 4, 5, 8
CONTAINS, NOT_CONTAINS = 99, 999
HAS_VALUE, NO_VALUE = 100, 101

ALL_INPUT_CLASSES = [
    "WFAppStoreAppContentItem", "WFArticleContentItem", "WFContactContentItem",
    "WFDateContentItem", "WFEmailAddressContentItem", "WFFolderContentItem",
    "WFGenericFileContentItem", "WFImageContentItem", "WFiTunesProductContentItem",
    "WFLocationContentItem", "WFDCMapsLinkContentItem", "WFAVAssetContentItem",
    "WFPDFContentItem", "WFPhoneNumberContentItem", "WFRichTextContentItem",
    "WFSafariWebPageContentItem", "WFStringContentItem", "WFURLContentItem",
]
MEDIA_INPUT_CLASSES = [
    "WFImageContentItem", "WFPhotoMediaContentItem", "WFAVAssetContentItem",
    "WFGenericFileContentItem", "WFPDFContentItem",
]
TEXT_INPUT_CLASSES = [
    "WFStringContentItem", "WFURLContentItem", "WFRichTextContentItem",
    "WFSafariWebPageContentItem",
]

# Icon colours from the Shortcuts palette.
BLUE, DARK_BLUE, TEAL, GREEN, GRAY = 463140863, 946986751, 431817727, 4292093695, 3031607807
GLYPH = 61440


# ---------------------------------------------------------------------------
# Values

def _aggs(key):
    if key is None:
        return {}
    return {"Aggrandizements": [
        {"Type": "WFCoercionVariableAggrandizement",
         "CoercionItemClass": "WFDictionaryContentItem"},
        {"Type": "WFDictionaryValueVariableAggrandizement", "DictionaryKey": key},
    ]}


def var(name, key=None):
    """A named variable, optionally one dictionary key of it (Config.secret)."""
    return dict({"Type": "Variable", "VariableName": name}, **_aggs(key))


def shortcut_input(key=None):
    return dict({"Type": "ExtensionInput"}, **_aggs(key))


CURRENT_DATE = {"Type": "CurrentDate"}
REPEAT_ITEM = "Repeat Item"


def utf16_len(s):
    return len(s.encode("utf-16-le")) // 2


def tok(*parts):
    """A text field: literal strings and variables, concatenated with nothing in between."""
    string, attachments = "", {}
    for p in parts:
        if isinstance(p, str):
            if OBJ in p:
                raise ValueError("literal text contains U+FFFC")
            string += p
        else:
            attachments["{%d, 1}" % utf16_len(string)] = p
            string += OBJ
    return {"Value": {"string": string, "attachmentsByRange": attachments},
            "WFSerializationType": "WFTextTokenString"}


def attach(ref):
    """A parameter that takes exactly one variable."""
    return {"Value": ref, "WFSerializationType": "WFTextTokenAttachment"}


def dict_field(items):
    """A Dictionary action's items / Get Contents of URL's headers. All values are Text."""
    out = []
    for key, value in items:
        parts = value if isinstance(value, (list, tuple)) else [value]
        out.append({"WFItemType": 0, "WFKey": tok(key), "WFValue": tok(*parts)})
    return {"Value": {"WFDictionaryFieldValueItems": out},
            "WFSerializationType": "WFDictionaryFieldValue"}


# ---------------------------------------------------------------------------
# Builder

class Shortcut:
    def __init__(self, name, color=BLUE, share_sheet=False, input_classes=None,
                 output_classes=None, no_input=None):
        self.name = name
        self.color = color
        self.share_sheet = share_sheet
        self.input_classes = input_classes or ALL_INPUT_CLASSES
        self.output_classes = output_classes or []
        self.no_input = no_input
        self.actions = []
        self._n = 0
        self._stack = []

    def _uuid(self):
        self._n += 1
        return str(uuid.uuid5(NS, "%s/%d" % (self.name, self._n))).upper()

    def act(self, ident, params=None, output=None):
        p = dict(params or {})
        u = self._uuid()
        p["UUID"] = u
        self.actions.append({"WFWorkflowActionIdentifier": "is.workflow.actions." + ident,
                             "WFWorkflowActionParameters": p})
        if output:
            return {"Type": "ActionOutput", "OutputUUID": u, "OutputName": output}
        return None

    # control flow -----------------------------------------------------------
    def if_(self, ref, condition, value=None, number=None):
        g = self._uuid()
        p = {"GroupingIdentifier": g, "WFControlFlowMode": 0, "WFCondition": condition,
             "WFInput": {"Type": "Variable", "Variable": attach(ref)}}
        if value is not None:
            p["WFConditionalActionString"] = value if isinstance(value, str) else tok(*value)
        if number is not None:
            p["WFNumberValue"] = number
        self.act("conditional", p)
        self._stack.append(["conditional", g, False])

    def otherwise(self):
        top = self._stack[-1]
        assert top[0] == "conditional" and not top[2], "Otherwise outside an If"
        top[2] = True
        self.act("conditional", {"GroupingIdentifier": top[1], "WFControlFlowMode": 1})

    def repeat_each(self, ref):
        g = self._uuid()
        self.act("repeat.each", {"GroupingIdentifier": g, "WFControlFlowMode": 0,
                                 "WFInput": attach(ref)})
        self._stack.append(["repeat.each", g, False])

    def end(self):
        kind, g, _ = self._stack.pop()
        self.act(kind, {"GroupingIdentifier": g, "WFControlFlowMode": 2})

    # actions ------------------------------------------------------------------
    def set(self, name, ref):
        self.act("setvariable", {"WFVariableName": name, "WFInput": attach(ref)})

    def text(self, *parts):
        return self.act("gettext", {"WFTextActionText": tok(*parts)}, "Text")

    def hash(self, ref):
        return self.act("hash", {"WFHashType": "SHA256", "WFInput": attach(ref)}, "Hash")

    def get_value(self, ref, key):
        return self.act("getvalueforkey", {"WFInput": attach(ref), "WFDictionaryKey": key,
                                           "WFGetDictionaryValueType": "Value"},
                        "Dictionary Value")

    def to_dictionary(self, ref):
        return self.act("detect.dictionary", {"WFInput": attach(ref)}, "Dictionary")

    def dictionary(self, items):
        return self.act("dictionary", {"WFItems": dict_field(items)}, "Dictionary")

    def alert(self, *parts):
        self.act("alert", {"WFAlertActionTitle": "PhoneDrop",
                           "WFAlertActionMessage": tok(*parts),
                           "WFAlertActionCancelButtonShown": False})

    def notify(self, *parts):
        self.act("notification", {"WFNotificationActionTitle": "PhoneDrop",
                                  "WFNotificationActionBody": tok(*parts),
                                  "WFNotificationActionSound": False})

    def stop(self):
        self.act("exit")

    def rename(self, ref, name):
        return self.act("setitemname", {"WFInput": attach(ref), "WFName": name,
                                        "WFDontIncludeFileExtension": False},
                        "Renamed Item")

    def save_to_shortcuts(self, ref, subpath):
        self.act("documentpicker.save", {"WFInput": attach(ref), "WFAskWhereToSave": False,
                                         "WFFileDestinationPath": subpath,
                                         "WFSaveFileOverwrite": True}, "Saved File")

    def sign(self, var_name, method, path, body, plain=False):
        """Dictionary {method, path, body[, plain]} -> Run "PhoneDrop Sign" -> var_name."""
        items = [("method", method), ("path", path), ("body", body)]
        if plain:
            items.append(("plain", "1"))
        d = self.dictionary(items)
        r = self.act("runworkflow", {
            "WFWorkflowName": SIGN_NAME,
            "WFWorkflow": {"workflowName": SIGN_NAME, "isSelf": False},
            "WFInput": attach(d),
            "WFShowWorkflow": False,
        }, "Shortcut Result")
        self.set(var_name, r)

    def request(self, req, method, body=None, extra_headers=()):
        """Get Contents of URL with Req.url and the five signed headers."""
        headers = [
            ("X-PD-Phone", var(req, "phone")),
            ("X-PD-Time", var(req, "time")),
            ("X-PD-Nonce", var(req, "nonce")),
            ("X-PD-Body", var(req, "body")),
            ("X-PD-Sig", var(req, "sig")),
        ] + list(extra_headers)
        p = {"WFURL": tok(var(req, "url")), "WFHTTPMethod": method, "ShowHeaders": True,
             "WFHTTPHeaders": dict_field(headers)}
        if body is not None:
            p["WFHTTPBodyType"] = "File"
            p["WFRequestVariable"] = attach(body)
        return self.act("downloadurl", p, "Contents of URL")

    # output -------------------------------------------------------------------
    def workflow(self):
        assert not self._stack, "%s: unclosed %s" % (self.name, self._stack)
        uses_input = any(_mentions_input(a) for a in self.actions)
        wf = {
            "WFWorkflowActions": self.actions,
            "WFWorkflowClientVersion": CLIENT_VERSION,
            "WFWorkflowMinimumClientVersion": 900,
            "WFWorkflowMinimumClientVersionString": "900",
            "WFWorkflowIcon": {"WFWorkflowIconStartColor": self.color,
                               "WFWorkflowIconGlyphNumber": GLYPH},
            "WFWorkflowImportQuestions": [],
            "WFWorkflowInputContentItemClasses": self.input_classes,
            "WFWorkflowOutputContentItemClasses": self.output_classes,
            "WFWorkflowTypes": ["ActionExtension"] if self.share_sheet else [],
            "WFQuickActionSurfaces": [],
            "WFWorkflowHasShortcutInputVariables": uses_input,
            "WFWorkflowHasOutputFallback": False,
        }
        if self.no_input:
            wf["WFWorkflowNoInputBehavior"] = self.no_input
        return wf


def _mentions_input(obj):
    if isinstance(obj, dict):
        if obj.get("Type") == "ExtensionInput":
            return True
        return any(_mentions_input(v) for v in obj.values())
    if isinstance(obj, list):
        return any(_mentions_input(v) for v in obj)
    return False


# ---------------------------------------------------------------------------
# The shortcuts (docs/SHORTCUTS.md)

def build_sign():
    """§1 + §7: returns {url, phone, time, nonce, body, sig} for one request."""
    s = Shortcut(SIGN_NAME, color=GRAY,
                 input_classes=ALL_INPUT_CLASSES + ["WFDictionaryContentItem"],
                 output_classes=["WFDictionaryContentItem"])
    f = s.act("documentpicker.open", {"WFGetFilePath": "PhoneDrop/config.json",
                                      "WFFileErrorIfNotFound": True,
                                      "WFShowFilePicker": False}, "File")
    s.set("ConfigFile", f)
    s.set("Config", s.to_dictionary(var("ConfigFile")))

    d = s.act("format.date", {"WFDate": tok(CURRENT_DATE),
                              "WFDateFormatStyle": "ISO 8601",
                              "WFISO8601IncludeTime": True}, "Formatted Date")
    s.set("Time", d)
    for name in ("R1", "R2"):
        r = s.act("number.random", {"WFRandomNumberMinimum": "100000000",
                                    "WFRandomNumberMaximum": "999999999"}, "Random Number")
        s.set(name, r)
    t = s.text(var("R1"), var("R2"), var("Time"))
    s.set("Nonce", s.hash(t))

    t = s.text(var("Config", "secret"), "|", shortcut_input("method"), "|",
               shortcut_input("path"), "|", var("Time"), "|", var("Nonce"), "|",
               shortcut_input("body"))
    s.set("Inner", s.hash(t))
    t = s.text(var("Config", "secret"), "|", var("Inner"))
    s.set("Sig", s.hash(t))

    # §7: https when the pairing has tls, except for requests that pass `plain`.
    s.set("Scheme", s.text("http"))
    tls = s.get_value(var("Config"), "tls")
    s.if_(tls, HAS_VALUE)
    plain = s.get_value(shortcut_input(), "plain")
    s.if_(plain, NO_VALUE)
    s.set("Scheme", s.text("https"))
    s.end()
    s.end()

    u = s.text(var("Scheme"), "://", var("Config", "base"), shortcut_input("path"))
    s.set("URL", u)
    out = s.dictionary([
        ("url", var("URL")),
        ("phone", var("Config", "id")),
        ("time", var("Time")),
        ("nonce", var("Nonce")),
        ("body", shortcut_input("body")),
        ("sig", var("Sig")),
    ])
    s.act("output", {"WFOutput": tok(out)})
    return s


def build_setup():
    """§2: scan the QR code, save it, pair, and (with tls) fetch the CA profile."""
    s = Shortcut("PhoneDrop Setup", color=DARK_BLUE)
    scan = s.act("scanbarcode", {}, "QR/Barcode")
    s.set("QR", s.text(scan))
    s.set("Pairing", s.to_dictionary(var("QR")))

    s.if_(s.get_value(var("Pairing"), "secret"), HAS_VALUE)
    s.save_to_shortcuts(s.rename(var("QR"), "config.json"), "PhoneDrop/config.json")

    dev = s.act("getdevicedetails", {"WFDeviceDetail": "Device Name"}, "Device Details")
    s.set("Name", dev)
    s.set("BodyHash", s.hash(var("Name")))
    s.sign("Req", "POST", "/v1/pair", var("BodyHash"), plain=True)
    s.set("Result", s.request("Req", "POST", body=var("Name")))
    s.if_(s.text(var("Result")), CONTAINS, "true")
    s.alert("Paired! You're ready to send.")
    s.otherwise()
    s.alert("Pairing didn't work: ", var("Result"))
    s.stop()
    s.end()

    # Encryption (§7): only when the QR code says tls.
    s.if_(s.get_value(var("Pairing"), "tls"), HAS_VALUE)
    s.sign("ProfileReq", "GET", "/v1/ca.mobileconfig", "-", plain=True)
    s.set("Profile", s.request("ProfileReq", "GET"))
    h = s.hash(var("Profile"))
    lower = s.act("text.changecase", {"WFCaseType": "lowercase", "text": tok(h)},
                  "Updated Text")
    s.set("ProfileHash", lower)
    s.if_(var("ProfileHash"), IS, [var("Pairing", "ca")])
    s.save_to_shortcuts(s.rename(var("Profile"), "PhoneDrop.mobileconfig"),
                        "PhoneDrop/PhoneDrop.mobileconfig")
    s.alert("Last step: in the Files app open iCloud Drive → Shortcuts → PhoneDrop "
            "and tap PhoneDrop.mobileconfig. Then Settings → Profile Downloaded → "
            "Install, and Settings → General → About → Certificate Trust "
            "Settings → turn on PhoneDrop.")
    s.otherwise()
    s.alert("The certificate didn't match the QR code. Don't install anything; "
            "pair again on your home network.")
    s.end()
    s.end()

    s.otherwise()
    s.alert("That isn't a PhoneDrop code.")
    s.end()
    return s


def send_each(s, items):
    """§3 step 1: upload every item in `items`, stop at the first failure."""
    s.repeat_each(items)
    item = var(REPEAT_ITEM)
    name = s.act("getitemname", {"WFInput": attach(item)}, "Name")
    enc = s.act("urlencode", {"WFInput": tok(name), "WFEncodeMode": "Encode"},
                "URL Encoded Text")
    s.set("EncodedName", enc)
    s.set("BodyHash", s.hash(item))
    s.sign("Req", "POST", "/v1/items", var("BodyHash"))
    s.set("Result", s.request("Req", "POST", body=item,
                              extra_headers=[("X-PD-Name", var("EncodedName"))]))
    s.if_(s.text(var("Result")), NOT_CONTAINS, "true")
    s.alert("PhoneDrop: ", var("Result"))
    s.stop()
    s.end()
    s.end()


def build_send():
    """§3: share sheet (images, media, files); with no input, pick photos."""
    s = Shortcut("Send to PC", color=BLUE, share_sheet=True,
                 input_classes=MEDIA_INPUT_CLASSES)
    # Apple's signing server rejects an If directly on Shortcut Input, so count it instead.
    s.set("Items", shortcut_input())
    s.set("InputCount", s.act("count", {"Input": attach(shortcut_input()), "WFCountType": "Items"},
                              "Count"))
    s.if_(var("InputCount"), IS, number="0")
    picked = s.act("selectphoto", {"WFSelectMultiplePhotos": True}, "Photos")
    s.set("Items", picked)
    s.end()
    send_each(s, var("Items"))
    s.notify("Sent to PC")
    return s


def build_send_text():
    """§4: share sheet (text, links, pages); with no input, the clipboard."""
    s = Shortcut("Send Text to PC", color=TEAL, share_sheet=True,
                 input_classes=TEXT_INPUT_CLASSES,
                 no_input={"Name": "WFWorkflowNoInputBehaviorGetClipboard",
                           "Parameters": {}})
    s.set("Body", s.text(shortcut_input()))
    s.set("BodyHash", s.hash(var("Body")))
    s.sign("Req", "POST", "/v1/items", var("BodyHash"))
    s.set("Result", s.request("Req", "POST", body=var("Body")))
    s.if_(s.text(var("Result")), NOT_CONTAINS, "true")
    s.alert("PhoneDrop: ", var("Result"))
    s.end()
    return s


def build_screenshot():
    """§5: Send to PC for the newest screenshot."""
    s = Shortcut("Send Latest Screenshot", color=BLUE)
    shots = s.act("getlastscreenshot", {"WFGetLatestPhotoCount": 1}, "Latest Screenshots")
    send_each(s, shots)
    s.notify("Sent to PC")
    return s


def build_receive():
    """§6: list the PC's outbox and collect every item."""
    s = Shortcut("Receive from PC", color=GREEN)
    s.sign("Req", "GET", "/v1/outbox", "-")
    s.set("List", s.request("Req", "GET"))
    lt = s.text(var("List"))
    s.set("ListText", lt)
    s.if_(var("ListText"), NOT_CONTAINS, "true")
    s.alert("PhoneDrop: ", var("List"))
    s.stop()
    s.end()

    listing = s.to_dictionary(var("ListText"))
    s.set("Items", s.get_value(listing, "items"))
    s.set("Count", s.act("count", {"Input": attach(var("Items")), "WFCountType": "Items"},
                         "Count"))
    s.if_(var("Count"), IS, number="0")
    s.notify("Nothing waiting on the PC")
    s.stop()
    s.end()

    s.repeat_each(var("Items"))
    item = var(REPEAT_ITEM)
    s.set("Kind", s.get_value(item, "kind"))

    # Apple's signing server rejects key lookups on Repeat Item and Open URLs fed plain text,
    # so read keys with Get Dictionary Value and turn links into a URL first.
    s.set("ItemText", s.get_value(item, "text"))
    s.set("ItemPath", s.get_value(item, "path"))

    s.if_(var("Kind"), IS, "text")
    s.act("setclipboard", {"WFInput": attach(var("ItemText"))})
    s.end()

    s.if_(var("Kind"), IS, "url")
    s.if_(var("ItemText"), BEGINS_WITH, "http")
    link = s.act("url", {"WFURLActionURL": tok(var("ItemText"))}, "URL")
    s.act("openurl", {"WFInput": attach(link)})
    s.otherwise()
    s.act("setclipboard", {"WFInput": attach(var("ItemText"))})
    s.end()
    s.end()

    for kind in ("image", "video", "file"):
        s.if_(var("Kind"), IS, kind)
        s.sign("FileReq", "GET", var("ItemPath"), "-")
        s.set("Download", s.request("FileReq", "GET"))
        if kind == "file":
            s.act("documentpicker.save", {"WFInput": attach(var("Download")),
                                          "WFAskWhereToSave": True}, "Saved File")
        else:
            s.act("savetocameraroll", {"WFInput": attach(var("Download")),
                                       "WFCameraRollSelectedGroup": "Recents"})
        s.end()
    s.end()
    s.notify("Received ", var("Count"), " from the PC")
    return s


BUILDERS = [build_sign, build_setup, build_send, build_send_text, build_screenshot,
            build_receive]


# ---------------------------------------------------------------------------
# Validator

KNOWN_ACTIONS = {
    "alert", "conditional", "count", "detect.dictionary", "dictionary",
    "documentpicker.open", "documentpicker.save", "downloadurl", "exit", "format.date",
    "getdevicedetails", "getitemname", "getlastscreenshot", "gettext", "getvalueforkey",
    "hash", "notification", "number.random", "openurl", "output", "repeat.each",
    "runworkflow", "savetocameraroll", "scanbarcode", "selectphoto", "setclipboard",
    "setitemname", "setvariable", "text.changecase", "urlencode", "url",
}
CONTROL_FLOW = {"conditional", "repeat.each"}
REF_TYPES = {"Variable", "ActionOutput", "ExtensionInput", "CurrentDate"}
AGG_TYPES = {"WFCoercionVariableAggrandizement", "WFDictionaryValueVariableAggrandizement"}
REQUIRED_TOP = {
    "WFWorkflowActions", "WFWorkflowClientVersion", "WFWorkflowMinimumClientVersion",
    "WFWorkflowMinimumClientVersionString", "WFWorkflowIcon", "WFWorkflowImportQuestions",
    "WFWorkflowInputContentItemClasses", "WFWorkflowOutputContentItemClasses",
    "WFWorkflowTypes", "WFWorkflowHasShortcutInputVariables",
}
RANGE_RE = re.compile(r"^\{(\d+), (\d+)\}$")


def _refs(obj, where, errors):
    """Yield every variable reference in obj, checking text-token ranges on the way."""
    if isinstance(obj, list):
        for v in obj:
            yield from _refs(v, where, errors)
        return
    if not isinstance(obj, dict):
        return
    kind = obj.get("WFSerializationType")
    if kind == "WFTextTokenString":
        value = obj.get("Value", {})
        string = value.get("string", "")
        units = string.encode("utf-16-le")
        atts = value.get("attachmentsByRange", {})
        if string.count(OBJ) != len(atts):
            errors.append("%s: %d U+FFFC but %d attachments in %r"
                          % (where, string.count(OBJ), len(atts), string))
        for rng, ref in atts.items():
            m = RANGE_RE.match(rng)
            if not m or m.group(2) != "1":
                errors.append("%s: bad range %r" % (where, rng))
                continue
            pos = int(m.group(1))
            if units[pos * 2:pos * 2 + 2].decode("utf-16-le", "replace") != OBJ:
                errors.append("%s: range %s is not at a U+FFFC in %r" % (where, rng, string))
            yield ref
        return
    if kind == "WFTextTokenAttachment":
        yield obj.get("Value", {})
        return
    for v in obj.values():
        yield from _refs(v, where, errors)


def validate(path):
    errors = []
    try:
        with open(path, "rb") as fh:
            wf = plistlib.load(fh)
    except Exception as e:  # noqa: BLE001 - report any parse failure
        return ["does not parse as a plist: %s" % e]
    missing = REQUIRED_TOP - set(wf)
    if missing:
        errors.append("missing top-level keys: %s" % ", ".join(sorted(missing)))
    has_input = wf.get("WFWorkflowHasShortcutInputVariables", False)
    stack, set_vars, seen_uuids = [], set(), set()
    for i, action in enumerate(wf.get("WFWorkflowActions", [])):
        ident = action.get("WFWorkflowActionIdentifier", "")
        short = ident.replace("is.workflow.actions.", "", 1)
        p = action.get("WFWorkflowActionParameters", {})
        where = "action %d (%s)" % (i + 1, short)
        if not ident.startswith("is.workflow.actions.") or short not in KNOWN_ACTIONS:
            errors.append("%s: unknown identifier %r" % (where, ident))

        for ref in _refs(p, where, errors):
            t = ref.get("Type")
            if t not in REF_TYPES:
                errors.append("%s: unknown reference type %r" % (where, t))
            if t == "Variable":
                name = ref.get("VariableName")
                if name == REPEAT_ITEM:
                    if not any(k == "repeat.each" for k, _, _ in stack):
                        errors.append("%s: Repeat Item outside a Repeat" % where)
                elif name not in set_vars:
                    errors.append("%s: variable %r used before it is set" % (where, name))
            elif t == "ActionOutput":
                if ref.get("OutputUUID") not in seen_uuids:
                    errors.append("%s: output of an unknown/later action" % where)
            elif t == "ExtensionInput" and not has_input:
                errors.append("%s: Shortcut Input used but HasShortcutInputVariables is off"
                              % where)
            for agg in ref.get("Aggrandizements", []):
                if agg.get("Type") not in AGG_TYPES:
                    errors.append("%s: unknown aggrandizement %r" % (where, agg.get("Type")))

        mode = p.get("WFControlFlowMode")
        if short in CONTROL_FLOW:
            g = p.get("GroupingIdentifier")
            if mode == 0:
                stack.append([short, g, False])
            elif mode == 1:
                if not stack or stack[-1][:2] != [short, g] or short != "conditional" \
                        or stack[-1][2]:
                    errors.append("%s: Otherwise does not match the open If" % where)
                else:
                    stack[-1][2] = True
            elif mode == 2:
                if not stack or stack[-1][:2] != [short, g]:
                    errors.append("%s: End does not match the open block" % where)
                else:
                    stack.pop()
            else:
                errors.append("%s: bad WFControlFlowMode %r" % (where, mode))
        elif mode is not None:
            errors.append("%s: WFControlFlowMode on a non-control-flow action" % where)

        if short == "setvariable":
            set_vars.add(p.get("WFVariableName"))
        if p.get("UUID"):
            if p["UUID"] in seen_uuids:
                errors.append("%s: duplicate UUID" % where)
            seen_uuids.add(p["UUID"])
    if stack:
        errors.append("unclosed blocks at the end: %s" % [k for k, _, _ in stack])
    return errors


# ---------------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--check", action="store_true", help="only validate shortcuts/unsigned/")
    ap.add_argument("--xml", metavar="DIR", help="also write XML plists into DIR")
    args = ap.parse_args()

    if not args.check:
        os.makedirs(OUT_DIR, exist_ok=True)
        for build in BUILDERS:
            s = build()
            wf = s.workflow()
            path = os.path.join(OUT_DIR, s.name + ".shortcut")
            with open(path, "wb") as fh:
                plistlib.dump(wf, fh, fmt=plistlib.FMT_BINARY, sort_keys=True)
            if args.xml:
                os.makedirs(args.xml, exist_ok=True)
                with open(os.path.join(args.xml, s.name + ".plist"), "wb") as fh:
                    plistlib.dump(wf, fh, fmt=plistlib.FMT_XML, sort_keys=True)

    names = sorted(n for n in os.listdir(OUT_DIR) if n.endswith(".shortcut")) \
        if os.path.isdir(OUT_DIR) else []
    if not names:
        print("no .shortcut files in %s" % OUT_DIR, file=sys.stderr)
        return 1
    failed = 0
    for n in names:
        errs = validate(os.path.join(OUT_DIR, n))
        if errs:
            failed += 1
            print("FAIL %s" % n)
            for e in errs:
                print("     %s" % e)
        else:
            with open(os.path.join(OUT_DIR, n), "rb") as fh:
                count = len(plistlib.load(fh)["WFWorkflowActions"])
            print("ok   %-28s %3d actions" % (n, count))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
