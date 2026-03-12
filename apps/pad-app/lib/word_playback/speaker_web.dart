import 'package:flutter_tts/flutter_tts.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker_contract.dart';

WordSpeaker createWordSpeaker() {
  return FlutterTtsSpeaker();
}

class FlutterTtsSpeaker implements WordSpeaker {
  FlutterTtsSpeaker() {
    _init();
  }

  Future<void>? _initFuture;
  final FlutterTts _flutterTts = FlutterTts();

  Future<void> _init() async {
    _initFuture = _doInit();
  }

  Future<void> _doInit() async {
    await _flutterTts.setSpeechRate(0.4);
    await _flutterTts.setVolume(1.0);
    await _flutterTts.setPitch(1.0);
    // Explicitly set a common language once at init to warm up
    await _flutterTts.setLanguage("en-US");
  }

  @override
  bool get supportsPlayback => true;

  @override
  Future<void> speak(
    String text, {
    required WordPlaybackLanguage language,
  }) async {
    if (_initFuture != null) await _initFuture;
    await _flutterTts.stop();
    await _flutterTts.setLanguage(language.localeCode);
    await _flutterTts.speak(text);
  }

  @override
  Future<void> stop() async {
    await _flutterTts.stop();
  }
}
