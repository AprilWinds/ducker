package image

import (
	"ducker/util"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
)

func Build(tag, duckerfilePath, contextPath string) error {
	parser := newDuckerfileParser(contextPath, duckerfilePath)
	if err := parser.parse(); err != nil {
		return fmt.Errorf("parse duckerfile: %w", err)
	}

	baseImg, err := resolveBaseImage(parser.getBaseImageTag())
	if err != nil {
		return fmt.Errorf("resolve base image: %w", err)
	}

	builder := NewBuilder(baseImg, tag, nil)
	if err := builder.Apply(parser.getInstructions()); err != nil {
		return fmt.Errorf("apply instructions: %w", err)
	}
	return builder.Build()
}

func Create(baseImageTag, newTag, newLayerPath string, runOpts *RunOptions) error {
	baseImage, err := resolveBaseImage(baseImageTag)
	if err != nil {
		return fmt.Errorf("resolve base image: %w", err)
	}

	builder := NewBuilder(baseImage, newTag, runOpts)
	if err := builder.CreateNewLayer(newLayerPath); err != nil {
		return fmt.Errorf("create new layer: %w", err)
	}
	return builder.Build()
}

// LoadBuiltin 加载内置镜像（从嵌入的 tar.gz 数据）
func LoadBuiltin(imageData []byte, tag string) error {
	if _, err := Get(tag); err == nil {
		return nil
	}

	tmpFile, err := os.CreateTemp("", "ducker-builtin-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(imageData); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	_, err = Load(tmpPath, tag)
	return err
}

func Load(archivePath, tag string) (*Image, error) {
	tag = normalizeTag(tag)
	imageID := util.GenerateID(tag)

	tempDir, err := os.MkdirTemp("", "ducker-import")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := util.ExtractArchive(archivePath, tempDir, true); err != nil {
		return nil, fmt.Errorf("extract image: %w", err)
	}

	imageDir := util.GetImageDir(imageID)
	os.RemoveAll(imageDir)
	if err := os.Rename(tempDir, imageDir); err != nil {
		return nil, fmt.Errorf("move image dir: %w", err)
	}

	return loadAndUpdateConfig(imageID, tag)
}

// loadAndUpdateConfig 加载配置并更新 tag 和 ID
func loadAndUpdateConfig(imageID, tag string) (*Image, error) {
	configData, err := os.ReadFile(util.GetImageConfigPath(imageID))
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var img Image
	if err := json.Unmarshal(configData, &img); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	img.Tag = tag
	img.ID = imageID
	if err := img.saveConfig(); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return &img, nil
}

func List(showAll, quiet bool) error {
	images, err := getAllImages()
	if err != nil {
		return fmt.Errorf("get images: %w", err)
	}

	slices.SortFunc(images, func(a, b *Image) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})

	if quiet {
		for _, img := range images {
			fmt.Println(img.Tag)
		}
		return nil
	}

	if !showAll {
		images = slices.DeleteFunc(images, func(img *Image) bool {
			return img.Hidden
		})
	}
	printImageInfo(images)
	return nil
}

func Get(tagOrID string) (*Image, error) {
	var imageID string
	if util.IsValidID(tagOrID) {
		imageID = tagOrID
	} else {
		imageID = util.GenerateID(normalizeTag(tagOrID))
	}

	img, err := util.FindBy[Image](util.TypeImage, imageID)
	if err != nil {
		return nil, fmt.Errorf("image %s not found", tagOrID)
	}
	return img, nil
}

// normalizeTag 标准化 tag，不带版本号时默认加 :latest
func normalizeTag(tag string) string {
	if !strings.Contains(tag, ":") {
		return tag + ":latest"
	}
	return tag
}

func GetRunOptions(tagOrID string) (*RunOptions, error) {
	img, err := Get(tagOrID)
	if err != nil {
		return nil, err
	}
	return img.RunOptions, nil
}

func GetLayers(tagOrID string) ([]string, error) {
	img, err := Get(tagOrID)
	if err != nil {
		return nil, err
	}
	return img.getLayers(), nil
}

func Save(imageTags []string, outputPath string) error {
	for _, tag := range imageTags {
		img, err := Get(tag)
		if err != nil {
			return fmt.Errorf("find image %s: %w", tag, err)
		}
		if err := img.save(outputPath); err != nil {
			return fmt.Errorf("save image %s: %w", tag, err)
		}
	}
	return nil
}

func Rm(imageTags []string, _ bool) error {
	for _, tag := range imageTags {
		img, err := Get(tag)
		if err != nil {
			return fmt.Errorf("find image %s: %w", tag, err)
		}
		if err := img.Remove(); err != nil {
			return fmt.Errorf("remove image %s: %w", tag, err)
		}
	}
	return nil
}

func resolveBaseImage(tag string) (*Image, error) {
	if img, err := Get(tag); err == nil {
		return img, nil
	}

	if tag != "alpine:latest" {
		return nil, fmt.Errorf("base image %q not found, only alpine:latest supports auto-import", tag)
	}
	return Load("./alpine.tar.gz", "alpine:latest")
}

func getAllImages() ([]*Image, error) {
	entries, err := os.ReadDir(util.GetImageRootDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []*Image{}, nil
		}
		return nil, fmt.Errorf("read image dir: %w", err)
	}

	var images []*Image
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if img, err := loadImageConfig(entry.Name()); err == nil {
			images = append(images, img)
		}
	}
	return images, nil
}

func loadImageConfig(imageID string) (*Image, error) {
	configData, err := os.ReadFile(util.GetImageConfigPath(imageID))
	if err != nil {
		return nil, err
	}
	var img Image
	if err := json.Unmarshal(configData, &img); err != nil {
		return nil, err
	}
	return &img, nil
}

func printImageInfo(images []*Image) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	fmt.Fprintln(writer, "IMAGE TAG\tIMAGE ID\tCREATED\tSIZE")
	for _, img := range images {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t\n",
			img.Tag, img.ID,
			util.FormatDuration(img.CreatedAt),
			util.FormatSize(img.Size),
		)
	}
}
