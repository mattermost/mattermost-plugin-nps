import {connected, userWantsToGiveFeedback} from './actions';

describe('connected', () => {
    const client = {
        connected: jest.fn(),
    };

    test('should call connected API if the user is logged in', () => {
        const getState = () => ({
            entities: {
                users: {
                    currentUserId: 'user1',
                },
            },
        });

        connected(client)(null, getState);

        expect(client.connected).toHaveBeenCalled();
    });

    test('should do nothing if the user is not logged in', () => {
        const getState = () => ({
            entities: {
                users: {
                    currentUserId: '',
                },
            },
        });

        connected(client)(null, getState);

        expect(client.connected).not.toHaveBeenCalled();
    });
});

describe('userWantsToGiveFeedback', () => {
    const client = {
        userWantsToGiveFeedback: jest.fn(),
    };

    test('should do nothing if the user is not logged in', () => {
        const getState = () => ({
            entities: {
                users: {
                    currentUserId: '',
                },
            },
        });

        userWantsToGiveFeedback(client)(null, getState);

        expect(client.userWantsToGiveFeedback).not.toHaveBeenCalled();
    });

    test('should call user wants feedback API if the user is logged in', () => {
        jest.mock('./browser_routing');
        jest.mock('mattermost-redux/selectors/entities/channels');

        const getState = () => ({
            entities: {
                users: {
                    currentUserId: 'user1',
                },
            },
        });
        const mockThen = jest.fn();
        client.userWantsToGiveFeedback.mockImplementationOnce(() => {
            return {then: mockThen};
        });

        userWantsToGiveFeedback(client)(null, getState);
        expect(client.userWantsToGiveFeedback).toHaveBeenCalled();
        expect(mockThen).toHaveBeenCalled();
    });
});
