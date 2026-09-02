package core

import (
	"blog/config"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// logEncoder 时间分片和level分片同时做
type logEncoder struct {
	zapcore.Encoder          //继承zapcore编码器
	errFile         *os.File //这里是日志要输出到的文件，一个errFile存储error级别日志，一个file存储所有级别日志
	file            *os.File
	currentDate     string //记录当前日期，实现按天记录日志，每当要输出一个日志，就检查currentDate和当前日期是否一致，如果不一致就创建新目录
}

// 这里相当于是截住了日志，做出手脚
func (e *logEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	/*首先是自定义日志*/
	// 先调用原始的 EncodeEntry 方法获取日志缓存buff，这个缓存buff里存有即将输出的日志
	buff, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}
	//获取日志
	data := buff.String()
	//清空缓存
	buff.Reset()
	//将这行日志进行修改，比如加上前缀[blog],当然了，这个前缀是从配置文件里读取的。除此之外，还可以加任何你想加的东西，这里不再赘述。
	//然后使用AppendStringb把处理好的日志塞进缓存
	buff.AppendString("[" + config.Cfg.LogConfig.App + "] " + data)
	data = buff.String()

	/*接下来是按时间存储日志文件*/
	// 将nowh和currentDate做比较，如果不一样，就代表该创建新目录了，如果一样，就跳过这段。
	now := time.Now().Format("2006-01-02")
	if e.currentDate != now {
		//
		if e.file != nil {
			err = e.file.Close()
			if err != nil {
				zap.L().Error("close file error", zap.Error(err))
			}
		}
		//检查目录
		err = os.MkdirAll(fmt.Sprintf(config.Cfg.LogConfig.Dir+"/%s", now), 0755)
		if err != nil {
			return nil, err
		}
		// 时间不同，先创建目录，并打开文件
		name := fmt.Sprintf(config.Cfg.LogConfig.Dir+"/%s/out.log", now)
		var file *os.File
		file, err = os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			return nil, err
		}
		e.file = file
		e.currentDate = now
	}

	//把err级别的日志存到err.log里。config.Cfg.LogConfig.Dir是从配置文件settings.yaml里读取的日志目录
	switch entry.Level {
	case zapcore.ErrorLevel:
		if e.errFile == nil {
			name := fmt.Sprintf(config.Cfg.LogConfig.Dir+"/%s/err.log", now)
			file, _ := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
			e.errFile = file
		}
		_, err = e.errFile.WriteString(data)
		if err != nil {
			return nil, err
		}
	default:
	}
	//再把所以级别的日志往out.log里存一份
	if e.currentDate == now {
		_, err2 := e.file.WriteString(data)
		if err2 != nil {
			return nil, err2
		}
	}
	//返回缓存
	return buff, nil
}

func LogInit() *zap.Logger {
	// 使用 zap 的 NewDevelopmentConfig 快速配置
	//先获取一个配置对象cfg，然后就可以修改许多配置，这里只修改了日志中日期的格式
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05") // 替换时间格式化方式
	//cfg.EncoderConfig.EncodeLevel = myEncodeLevel
	//创建 logEncoder对象，并把继承的默认编码器(Encoder)换为刚刚自定义的cfg.EncoderConfig
	encoder := &logEncoder{
		Encoder: zapcore.NewConsoleEncoder(cfg.EncoderConfig), // 使用 Console 编码器，自定义分片
	}
	// 读取settings.yaml里的日志级别level
	var level zapcore.Level = zapcore.DebugLevel
	switch config.Cfg.LogConfig.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "panic":
		level = zapcore.PanicLevel
	case "dpanic":
		level = zapcore.DPanicLevel
	case "fatal":
		level = zapcore.FatalLevel
	}
	//创建 Core，之前的配置都会体现在这里
	core := zapcore.NewCore(
		encoder,                                // 使用自定义的 编码器（Encoder），
		zapcore.NewMultiWriteSyncer(os.Stdout), // 输出到控制台，在自定义的方法里，日志已经存到了文件里，所有这里再写一个输出到控制台就行了
		level,                                  // 设置日志级别
	)
	// 创建 Logger
	logger := zap.New(core, zap.AddCaller()) //zap.AddCaller()可以让输出的日志多出：调用该日志的文件和行号，帮助快速找到问题所在

	zap.ReplaceGlobals(logger) // 替换全局Logger实例的函数，通过这个，你可以在任何地方使用zap.L()调用同一个Logger实例，而无需创建日志实例
	return logger
}
