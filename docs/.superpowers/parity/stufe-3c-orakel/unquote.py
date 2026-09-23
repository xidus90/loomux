from urllib.parse import unquote

cases = ["%e2%82", "%e0%a0", "%e0%80", "%ed%a0", "%ed%9f", "%f0%90%80", "%f0%80", "%f4%90", "%f4%8f",
         "%f1%80", "%c2", "%c0%af", "%80x", "%e2%82%ac", "%e1%80%41", "%ff%fe"]
for c in cases:
    print(c, ascii(unquote(c)))
