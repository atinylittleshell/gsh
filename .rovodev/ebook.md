Create TODO items tracking the top level bullet points below.

- Read `spec/GSH_LANG_EBOOK.md` to find the chapter plan
- Identify the next chapter marked as TODO - that's the one for you to write. And only need to write this one.
- Read the entirety of `spec/GSH_SCRIPT_SPEC.md` (line range [0, -1]) and relevant code files for language details needed for the chapter
- Write the chapter following the structure: opening → core concepts → examples → key takeaways → what's next
  - Use conversational tone, concrete examples before abstraction, active voice
  - Include output/results for all examples
  - Link to previous/next chapters with relative markdown links
  - Call create_file tool with overwrite=true to write the chapter file
- Every code example must be complete, tested, and actually runnable in gsh
- Mark the chapter as DONE in `spec/GSH_LANG_EBOOK.md` when finished
- If you discovered any real bugs in our language implementation, report them in spec/BUGS.md
