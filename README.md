# Convert Mobiledoc articles to Markdown

For example, given a Ghost blog export file, you can grab an article
and convert it to Markdown like this:

```
jq -r '.db[0].data.posts[] | select(.id == "ghostblogarticleidhere42")' my-ghost-blog.2019-02-17.json >article.json
mobiledoc-to-markdown article.json >article.md
```
