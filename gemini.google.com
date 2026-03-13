(myenv)  ✘ hmza@0root  ~/workspaces/hamza/LeakPeek   main  ./leakpeek "https://gemini.google.com/" workers:10 depth:10 "key:AIza[A-Za-z0-9_-]{35}" "aws:AKIA[A-Z0-9]{16}" refmt:25-25
[LeakPeek] starting → https://gemini.google.com   depth:10   workers:10   context:25-25
[format] timestamp      rule    url     "match" context
---------------------------------------------------------------
2026-03-13T18:20:16Z    key     https://gemini.google.com       "AIzaSyCqyCcs2R2e7AegGjvFAwG98wlamtbHvZY"       \\n  \\\"apiKey\\\": \\\"AIzaSyCqyCcs2R2e7AegGjvFAwG98wlamtbHvZY\\\",\\n  \\\"authDomain\
2026-03-13T18:20:16Z    key     https://gemini.google.com       "AIzaSyD6n9asBjvx1yBHfhFhfw_kpS9Faq0BZHM"       the occasion.","VVlN6d":"AIzaSyD6n9asBjvx1yBHfhFhfw_kpS9Faq0BZHM","Vvafkd":false,"Wv9gkb"
2026-03-13T18:20:16Z    key     https://gemini.google.com       "AIzaSyAPW83vB9zFQqfpMup_cMJdELqDQkWvTho"       0260311.02_p5","d2zJAe":"AIzaSyAPW83vB9zFQqfpMup_cMJdELqDQkWvTho","dLc0B":false,"eptZe":"
2026-03-13T18:20:16Z    key     https://gemini.google.com       "AIzaSyBWW50ghQ5qHpMg1gxHV7U9t0wHE0qIUk4"       ,"hwjqj":false,"i1PRRd":"AIzaSyBWW50ghQ5qHpMg1gxHV7U9t0wHE0qIUk4","iCzhFc":false,"kmz9uc"
2026-03-13T18:20:16Z    key     https://gemini.google.com       "AIzaSyDmUQ6sj3nbs_ZiSsxsbP7L6qlPDT3cr4Q"       ,"mXaIFf":true,"nPMdNb":"AIzaSyDmUQ6sj3nbs_ZiSsxsbP7L6qlPDT3cr4Q","nQyAE":{"u2B5ld":"true
2026-03-13T18:20:16Z    key     https://gemini.google.com       "AIzaSyAHCfkEDYwQD6HuUx2DyX3VylTrKZG7doM"       instructions.","ypY7lb":"AIzaSyAHCfkEDYwQD6HuUx2DyX3VylTrKZG7doM","zChJod":"%.@.]","zEdwB
2026-03-13T18:20:19Z    key     https://gemini.google.com/      "AIzaSyCqyCcs2R2e7AegGjvFAwG98wlamtbHvZY"       \\n  \\\"apiKey\\\": \\\"AIzaSyCqyCcs2R2e7AegGjvFAwG98wlamtbHvZY\\\",\\n  \\\"authDomain\
2026-03-13T18:20:19Z    key     https://gemini.google.com/      "AIzaSyD6n9asBjvx1yBHfhFhfw_kpS9Faq0BZHM"       the occasion.","VVlN6d":"AIzaSyD6n9asBjvx1yBHfhFhfw_kpS9Faq0BZHM","Vvafkd":false,"Wv9gkb"
2026-03-13T18:20:19Z    key     https://gemini.google.com/      "AIzaSyAPW83vB9zFQqfpMup_cMJdELqDQkWvTho"       0260311.02_p5","d2zJAe":"AIzaSyAPW83vB9zFQqfpMup_cMJdELqDQkWvTho","dLc0B":false,"eptZe":"
2026-03-13T18:20:19Z    key     https://gemini.google.com/      "AIzaSyBWW50ghQ5qHpMg1gxHV7U9t0wHE0qIUk4"       ,"hwjqj":false,"i1PRRd":"AIzaSyBWW50ghQ5qHpMg1gxHV7U9t0wHE0qIUk4","iCzhFc":false,"kmz9uc"
2026-03-13T18:20:19Z    key     https://gemini.google.com/      "AIzaSyDmUQ6sj3nbs_ZiSsxsbP7L6qlPDT3cr4Q"       ,"mXaIFf":true,"nPMdNb":"AIzaSyDmUQ6sj3nbs_ZiSsxsbP7L6qlPDT3cr4Q","nQyAE":{"u2B5ld":"true
2026-03-13T18:20:19Z    key     https://gemini.google.com/      "AIzaSyAHCfkEDYwQD6HuUx2DyX3VylTrKZG7doM"       instructions.","ypY7lb":"AIzaSyAHCfkEDYwQD6HuUx2DyX3VylTrKZG7doM","zChJod":"%.@.]","zEdwB
^C
