Searching:
Search by artist name and related artists (as determined by EchoNest),
possibly with several songs by that artist (if provided), will be displayed.

Results are currently limited to avoid flooding EchoNest API while testing.


Library:
Place music files in a dir named 'music'.

On start all of the files in this folder will have their ID3 information
grabbed, and placed into the database under the table 'songs'.


TODOS:
Create artist->song relation, want to move to GORM but postgresql setup is not working with it currently

Finish frontend for library management