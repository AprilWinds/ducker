package image

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// instruction 表示一个duckerfile指令
type instruction struct {
	command string
	args    []string
	raw     string
}

// 支持的命令列表
var supportedCommands = map[string]bool{
	"FROM": true, "RUN": true, "ENV": true, "WORKDIR": true,
	"EXPOSE": true, "CMD": true, "COPY": true,
}

func isCommandSupported(command string) bool {
	return supportedCommands[command]
}

type duckerfileParser struct {
	baseImageTag string
	instructions []*instruction
	lineRegex    *regexp.Regexp
	contextPath  string
	filepath     string
}

func newDuckerfileParser(contextPath, duckerfilePath string) *duckerfileParser {
	return &duckerfileParser{
		instructions: make([]*instruction, 0, 10),
		lineRegex:    regexp.MustCompile(`^(\w+)\s+(.*)$`),
		contextPath:  contextPath,
		filepath:     duckerfilePath,
	}
}

func (dp *duckerfileParser) parse() error {
	if dp.contextPath != "" {
		dp.filepath = filepath.Join(dp.contextPath, dp.filepath)
	}

	file, err := os.Open(dp.filepath)
	if err != nil {
		return fmt.Errorf("open duckerfile: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		instruction, err := dp.parseLine(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}

		// 记录基础镜像
		if instruction.command == "FROM" && len(instruction.args) > 0 {
			dp.baseImageTag = instruction.args[0]
		}
		dp.instructions = append(dp.instructions, instruction)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan duckerfile: %w", err)
	}

	slog.Info("duckerfile parsed successfully",
		slog.String("baseImage", dp.baseImageTag),
		slog.Int("instructions", len(dp.instructions)))

	return nil
}

func (dp *duckerfileParser) parseLine(line string) (*instruction, error) {
	matches := dp.lineRegex.FindStringSubmatch(line)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid instruction format: %s", line)
	}

	command := strings.ToUpper(matches[1])
	argsStr := strings.TrimSpace(matches[2])

	if !isCommandSupported(command) {
		return nil, fmt.Errorf("unsupported instruction: %s", command)
	}

	args, err := dp.parseCommand(command, argsStr)
	if err != nil {
		return nil, fmt.Errorf("parse command %s: %w", command, err)
	}
	return &instruction{
		command: command,
		args:    args,
		raw:     line,
	}, nil
}

func (dp *duckerfileParser) parseCommand(command, argsStr string) ([]string, error) {
	switch command {
	case "CMD":
		return dp.parseCMDArgs(argsStr)
	case "ENV":
		return dp.parseEnvArgs(argsStr)
	case "COPY":
		return dp.parseCopyArgs(argsStr)
	case "EXPOSE":
		return strings.Fields(argsStr), nil
	default:
		return []string{argsStr}, nil
	}
}

func (dp *duckerfileParser) parseCMDArgs(argsStr string) ([]string, error) {
	argsStr = strings.TrimSpace(argsStr)
	// exec格式: ["cmd", "arg1", "arg2"]
	if strings.HasPrefix(argsStr, "[") && strings.HasSuffix(argsStr, "]") {
		argsStr = strings.Trim(argsStr, "[]")
		parts := strings.Split(argsStr, ",")
		args := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, `'"`)
			if part != "" {
				args = append(args, part)
			}
		}
		return args, nil
	}

	return nil, fmt.Errorf("missing exec format [ ]")
}

func (dp *duckerfileParser) parseCopyArgs(argsStr string) ([]string, error) {
	parts := strings.Split(argsStr, " ")
	if len(parts) != 2 {
		return []string{argsStr}, fmt.Errorf("invalid copy args")
	}
	if filepath.IsAbs(parts[0]) {
		return parts, nil
	}
	parts[0] = filepath.Join(dp.contextPath, parts[0])
	return parts, nil
}

func (dp *duckerfileParser) parseEnvArgs(argsStr string) ([]string, error) {
	parts := strings.Fields(argsStr)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, "=") {
			return nil, fmt.Errorf("part %s invalid", part)
		}
	}
	return parts, nil
}

func (dp *duckerfileParser) getInstructions() []*instruction {
	return dp.instructions
}

func (dp *duckerfileParser) getBaseImageTag() string {
	return dp.baseImageTag
}
