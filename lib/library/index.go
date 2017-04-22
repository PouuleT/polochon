package library

import (
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

// RebuildIndex rebuilds both the movie and show index
func (l *Library) RebuildIndex(log *logrus.Entry) error {
	// Create a goroutine for each index
	var wg sync.WaitGroup
	errc := make(chan error, 2)
	wg.Add(2)

	// Build the movie index
	l.movieIndex.Clear()
	go func() {
		defer wg.Done()
		if err := l.buildMovieIndex(log); err != nil {
			errc <- err
		}
	}()

	// Build the show index
	l.showIndex.Clear()
	go func() {
		defer wg.Done()
		if err := l.buildShowIndex(log); err != nil {
			errc <- err
		}
	}()

	// Wait for them to be done
	wg.Wait()
	close(errc)

	// Return the first error found
	err, ok := <-errc
	if ok {
		return err
	}

	return nil
}

func (l *Library) buildMovieIndex(log *logrus.Entry) error {
	start := time.Now()
	err := filepath.Walk(l.MovieDir, func(filePath string, file os.FileInfo, err error) error {
		// Check err
		if err != nil {
			log.Errorf("library: failed to walk %q", err)
			return nil
		}

		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for movie type
		ext := path.Ext(filePath)

		var moviePath string
		for _, mext := range l.fileConfig.VideoExtentions {
			if ext == mext {
				moviePath = filePath
				break
			}
		}

		if moviePath == "" {
			return nil
		}

		// Read the movie informations
		movie, err := l.newMovieFromPath(moviePath)
		if err != nil {
			log.Errorf("library: failed to read movie NFO: %q", err)
			return nil
		}

		// Add the movie to the index
		l.movieIndex.Add(movie)

		return nil
	})

	log.Infof("Index built in %s", time.Since(start))

	return err
}

func (l *Library) buildShowIndex(log *logrus.Entry) error {
	start := time.Now()

	// used to catch if the first root folder has been walked
	var rootWalked bool
	// Get only the parent folders
	err := filepath.Walk(l.ShowDir, func(filePath string, file os.FileInfo, err error) error {
		// Only check directories
		if !file.IsDir() {
			return nil
		}

		// The root folder is only walk once
		if !rootWalked {
			rootWalked = true
			return nil
		}

		// Check if we can find the tvshow.nfo file
		nfoPath := l.showNFOPath(filePath)
		show, err := l.newShowFromPath(nfoPath)
		if err != nil {
			log.Errorf("library: failed to read tv show NFO: %q", err)
			return nil
		}

		// Scan the path for the episodes
		err = l.scanEpisodes(show.ImdbID, filePath, log)
		if err != nil {
			return err
		}

		// No need to go deeper, the tvshow.nfo is in the second root folder
		return filepath.SkipDir
	})
	if err != nil {
		return err
	}

	log.Infof("Index built in %s", time.Since(start))

	return nil

}

func (l *Library) scanEpisodes(imdbID, showRootPath string, log *logrus.Entry) error {
	// Walk the files of a show
	err := filepath.Walk(showRootPath, func(filePath string, file os.FileInfo, err error) error {
		// Check err
		if err != nil {
			log.Errorf("library: failed to walk %q", err)
			return nil
		}

		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for show type
		ext := path.Ext(filePath)

		var epPath string
		for _, mext := range l.fileConfig.VideoExtentions {
			if ext == mext {
				epPath = filePath
				break
			}
		}

		if epPath == "" {
			return nil
		}

		// Read the nfo file
		episode, err := l.newEpisodeFromPath(epPath)
		if err != nil {
			log.Errorf("library: failed to read episode NFO: %q", err)
			return nil
		}

		episode.ShowImdbID = imdbID
		episode.ShowConfig = l.showConfig
		l.showIndex.Add(episode)

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
