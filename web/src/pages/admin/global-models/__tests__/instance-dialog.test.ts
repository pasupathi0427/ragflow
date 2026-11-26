import { toRequestModelType } from '../constants';
import { toModelInputs } from '../instance-dialog';

jest.mock('../hooks', () => ({
  useCreateGlobalInstance: jest.fn(),
  useTestGlobalConnection: jest.fn(),
}));

describe('toModelInputs', () => {
  it('sends the type names model verification understands', () => {
    const inputs = toModelInputs([
      { key: 1, name: ' qwen3-vl ', types: ['image2text'], maxTokens: 8192 },
      { key: 2, name: 'whisper', types: ['speech2text'], maxTokens: 0 },
      { key: 3, name: 'gpt', types: ['chat', 'image2text'], maxTokens: 4096 },
    ]);

    expect(inputs).toEqual([
      { model_name: 'qwen3-vl', model_type: ['vision'], max_tokens: 8192 },
      { model_name: 'whisper', model_type: ['asr'], max_tokens: 0 },
      { model_name: 'gpt', model_type: ['chat', 'vision'], max_tokens: 4096 },
    ]);
  });

  it('drops models without a name or a type', () => {
    expect(
      toModelInputs([
        { key: 1, name: '  ', types: ['chat'], maxTokens: 1 },
        { key: 2, name: 'm', types: [], maxTokens: 1 },
      ]),
    ).toEqual([]);
  });
});

describe('toRequestModelType', () => {
  it('keeps names that are the same in both forms', () => {
    expect(toRequestModelType('chat')).toBe('chat');
    expect(toRequestModelType('embedding')).toBe('embedding');
    expect(toRequestModelType('rerank')).toBe('rerank');
  });
});
