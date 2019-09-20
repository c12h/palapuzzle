# palapuzzle
Go package for examining .puzzle files from the KDE Jigsaw program Palapeli

Palapeli is a KDE app for creating and solving Jigsaw puzzles, which are gzipped tarballs usually named $TITLE.puzzle. This package exports one function, ScanPuzzle(),
which returns (a struct containing) details of a .puzzle file.
