import 'package:pad_app/voice_commands/speech_recognizer_contract.dart';
import 'package:pad_app/voice_commands/speech_recognizer_impl.dart'
    as implementation;

SpeechRecognizer createSpeechRecognizer() {
  return implementation.createSpeechRecognizer();
}
