// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useSurveyPost} from './selectors';

import Plugin from './index';

jest.mock('./selectors');

describe('Plugin', () => {
    describe('registerSurveyPost', () => {
        test('should register score post with enough space available', () => {
            const plugin = new Plugin();
            plugin.registry = {
                registerPostTypeComponent: jest.fn(() => 'componentId'),
            };

            useSurveyPost.mockReturnValue(true);

            plugin.registerSurveyPost({});

            expect(plugin.registry.registerPostTypeComponent).toHaveBeenCalled();
            expect(plugin.overrideSurveyPost).toBe(true);
            expect(plugin.surveyPostComponentId).toBe('componentId');
        });

        test('should not register score post without enough space available', () => {
            const plugin = new Plugin();
            plugin.registry = {
                registerPostTypeComponent: jest.fn(() => 'componentId'),
            };

            useSurveyPost.mockReturnValue(false);

            plugin.registerSurveyPost({});

            expect(plugin.registry.registerPostTypeComponent).not.toHaveBeenCalled();
            expect(plugin.overrideSurveyPost).toBe(false);
            expect(plugin.surveyPostComponentId).toBe('');
        });

        test('should register and unregister score post as size changes', () => {
            const plugin = new Plugin();
            plugin.registry = {
                registerPostTypeComponent: jest.fn(() => 'componentId'),
                unregisterPostTypeComponent: jest.fn(),
            };

            useSurveyPost.mockReturnValue(true);

            plugin.registerSurveyPost({});

            expect(plugin.registry.registerPostTypeComponent).toHaveBeenCalledTimes(1);
            expect(plugin.overrideSurveyPost).toBe(true);
            expect(plugin.surveyPostComponentId).toBe('componentId');

            useSurveyPost.mockReturnValue(false);

            plugin.registerSurveyPost({});

            expect(plugin.registry.unregisterPostTypeComponent).toHaveBeenCalledTimes(1);
            expect(plugin.overrideSurveyPost).toBe(false);
            expect(plugin.surveyPostComponentId).toBe('');

            useSurveyPost.mockReturnValue(true);

            plugin.registerSurveyPost({});

            expect(plugin.registry.registerPostTypeComponent).toHaveBeenCalledTimes(2);
            expect(plugin.overrideSurveyPost).toBe(true);
            expect(plugin.surveyPostComponentId).toBe('componentId');
        });
    });
});
