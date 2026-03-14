import 'package:flutter/services.dart';
import 'package:flutter_tts/flutter_tts.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker_contract.dart';

WordSpeaker createWordSpeaker() {
  return FlutterTtsSpeaker();
}

class FlutterTtsSpeaker implements WordSpeaker {
  static const double _defaultSpeechRate = 0.4;
  static const double _defaultPitch = 1.0;

  Future<void>? _initFuture;
  final FlutterTts _flutterTts = FlutterTts();
  bool _pluginAvailable = true;

  Future<void> _ensureInitialized() {
    return _initFuture ??= _doInit();
  }

  Future<void> _doInit() async {
    await _guardPluginCall(() => _flutterTts.setSpeechRate(_defaultSpeechRate));
    await _guardPluginCall(() => _flutterTts.setVolume(1.0));
    await _guardPluginCall(() => _flutterTts.setPitch(_defaultPitch));
    // Warm up a common locale once so later language switches are faster.
    await _guardPluginCall(() => _flutterTts.setLanguage('en-US'));
  }

  @override
  bool get supportsPlayback => _pluginAvailable;

  @override
  Future<void> speak(
    String text, {
    required WordPlaybackLanguage language,
    double? speechRate,
    double? pitch,
  }) async {
    await _ensureInitialized();
    if (!_pluginAvailable) {
      return;
    }

    await _guardPluginCall(_flutterTts.stop);
    await _guardPluginCall(
        () => _flutterTts.setSpeechRate(speechRate ?? _defaultSpeechRate));
    await _guardPluginCall(() => _flutterTts.setPitch(pitch ?? _defaultPitch));
    await _guardPluginCall(() => _flutterTts.setLanguage(language.localeCode));
    await _guardPluginCall(() => _flutterTts.speak(text));
  }

  @override
  Future<void> stop() async {
    await _guardPluginCall(_flutterTts.stop);
  }

  Future<void> _guardPluginCall(Future<dynamic> Function() action) async {
    try {
      await action();
    } on MissingPluginException {
      _pluginAvailable = false;
    }
  }
}
